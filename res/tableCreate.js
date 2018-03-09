(function () {
    function copyRows($checkboxes) {
        $checkboxes.each(function (index, checkbox) {
            var $tr = $(checkbox).parents('tr')
            $tr.find(':checked').prop("checked", false)
            var $clone = $tr.clone().addClass('clonedRow')
            $clone.insertAfter($tr)
            $clone.find('input[type=checkbox]').click($.toggleRowEditable).click()
        })
    }

    function attachDeleteRowsEvent(queryResultId) {
        var cssChoser = '#queryResult' + queryResultId + ' :checked'
        $('#deleteRows' + queryResultId).click(function () {
            $(cssChoser).parents('tr').addClass('deletedRow')
        })
    }

    function attachCopyRowsEvent(thisQueryResult) {
        $('#copyRow' + thisQueryResult).click(function () {
            var checkboxes = $('#queryResult' + thisQueryResult + ' :checked')
            if (checkboxes.length == 0) {
                alert('please specify which row to copy')
            } else {
                copyRows($(checkboxes))
            }
        })
    }

    function createTenantMap(tenants) {
        var tenantsMap = {}
        for (var i = 0; i < tenants.length; ++i) {
            var tenant = tenants[i]
            tenantsMap[tenant.merchantId] = tenant
        }
        return tenantsMap
    }

    function createTenantIdGroup(tenants, groupSize) {
        var tenantIdsGroup = []
        var group = []
        for (var i = 0; i < tenants.length; ++i) {
            group.push(tenants[i].merchantId)

            if (group.length == groupSize) {
                tenantIdsGroup.push(group)
                group = []
            }
        }

        if (group.length > 0) {
            tenantIdsGroup.push(group)
        }

        return tenantIdsGroup;
    }

    function attachOpsResultDivEvent(resultId) {
        var divId = '#executionResultDiv' + resultId
        $('#closeResult' + resultId).click(function () {
            $(divId).remove()
        })

        $('#reExecuteSql' + resultId).click(function () {
            var tid = $(this).attr('tid')
            var tname = $(this).attr('tname')
            var sql = $(divId).find('.sqlTd').text()
            $.executeQueryAjax(tid, tname, sql, resultId)
        })

        var multipleTenantsExecutable = $('#multipleTenantsExecutable' + resultId);
        multipleTenantsExecutable.find('.opsSpan').click(function () {
            var sql = $.trim($.getEditorSql())
            if (sql === "") {
                alert("please input the sql!")
                return
            }

            var $this = $(this)

            var merchantIdIndex = parseInt($this.attr('merchantIdIndex'))
            var merchantNameIndex = parseInt($this.attr('merchantNameIndex'))
            var merchantCodeIndex = parseInt($this.attr('merchantCodeIndex'))
            var tenants = findTenants(resultId, merchantIdIndex, merchantNameIndex, merchantCodeIndex)

            var batchSizeInput = multipleTenantsExecutable.find('.batchSize');
            var batchSize = parseInt(batchSizeInput.val() || batchSizeInput.prop('placeholder'))
            var tenantIdsGroup = createTenantIdGroup(tenants, batchSize)

            if (tenantIdsGroup.length > 0) {
                var tenantsMap = createTenantMap(tenants)
                var batchConfirm = multipleTenantsExecutable.find('.confirm').prop('checked')
                multipleTenantsQueryAjax(sql, tenantsMap, ++queryResultId, 0, tenantIdsGroup, 0, 0, Date.now(), batchConfirm)
            }
        })
    }

    function sortContent(content, currentGroupIds, headerHolder, groupIndex) {
        var resortedCotent = []

        for (var i = 0; i < currentGroupIds.length; ++i) {
            for (var j = 0; j < content.length; ++j) {
                if (groupIndex == 0 && content[j].Headers && content[j].Headers.length) {
                    headerHolder.Headers = content[j].Headers
                }

                if (content[j].Tid === currentGroupIds[i]) {
                    resortedCotent.push(content[j])
                    break
                }
            }
        }

        return resortedCotent
    }

    function multipleTenantsQueryAjax(sql, tenantsMap, resultId, groupIndex, tenantIdsGroup, headerColumnsLen, dataRowsIndex, startTime, batchConfirm) {
        if (groupIndex >= tenantIdsGroup.length || (groupIndex > 0 && batchConfirm && !window.confirm('Continue?'))) {
            $('#queryResult' + resultId + ' tbody tr:odd').addClass('rowOdd').attr('rowOdd', true)
            $.attachSearchTableEvent(resultId, 0)
            attachExpandRowsEvent(resultId)
            attachOpsResultDivEvent(resultId)

            $('#summaryRows' + resultId).text(dataRowsIndex)
            $('#summaryCostTime' + resultId).text($.costTime(startTime))
            $.createOrderByContextMenu(resultId)

            return
        }

        var currentGroup = tenantIdsGroup[groupIndex]
        var multipleTenantIds = currentGroup.join(',')
        $.ajax({
            type: 'POST',
            url: pathname + "/multipleTenantsQuery",
            data: {multipleTenantIds: multipleTenantIds, sql: sql},
            success: function (content, textStatus, request) {
                if (content && content.length === 1 && content[0].Tid === "" && content[0].Error !== "") {
                    // Maybe :
                    // [{"Headers":null,"Rows":null,"Error":"dangerous sql, please get authorized first!","ExecutionTime":"2018-03-09 13:41:06.443",
                    // "CostTime":"8.591µs","DatabaseName":"","TableName":"","PrimaryKeysIndex":null,"Msg":"","Tid":""}]
                    alert(content[0].Error)
                    return
                }

                var headerHolder = {}
                var resortedContent = sortContent(content, currentGroup, headerHolder, groupIndex)
                if (groupIndex == 0) {
                    tableCreateSimpleHead(headerHolder.Headers, sql, resultId)
                    headerColumnsLen = headerHolder.Headers ? headerHolder.Headers.length : 1
                }

                var rows = ''
                for (var i = 0; i < resortedContent.length; ++i) {
                    rows += $.createRowsSimple(tenantsMap, resortedContent[i], headerColumnsLen, dataRowsIndex)
                    dataRowsIndex += resortedContent[i].Rows && resortedContent[i].Rows.length ? resortedContent[i].Rows.length : 1
                }

                $('#queryResult' + resultId + " tbody").append(rows)

                setTimeout(function () { // Leave time for rendering, and then continue to next batch.
                    multipleTenantsQueryAjax(sql, tenantsMap, resultId, groupIndex + 1, tenantIdsGroup, headerColumnsLen, dataRowsIndex, startTime, batchConfirm)
                }, 100)
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
            }
        })
    }

    var findTenants = function (queryResultId, merchantIdIndex, merchantNameIndex, merchantCodeIndex) {
        var checkboxEditable = $('#checkboxEditable' + queryResultId).prop('checked')
        var chosenRows = checkboxEditable
            ? $('#queryResult' + queryResultId + ' :checked').parents('tr:visible')
            : $('#queryResult' + queryResultId + ' tr.dataRow:visible')

        var offset = 2
        var tenants = []
        chosenRows.each(function (index, tr) {
            var tds = $(tr).find('td');
            var item = {
                merchantId: tds.eq(merchantIdIndex + offset).text(),
                merchantName: tds.eq(merchantNameIndex + offset).text(),
                merchantCode: tds.eq(merchantCodeIndex + offset).text()
            }
            if (checkboxEditable) {
                tenants.unshift(item)
            } else {
                tenants.push(item)
            }
        })

        return tenants
    }

    function attachExpandRowsEvent(queryResultId) {
        var buttonId = '#expandRows' + queryResultId
        var collapseDiv = '#collapseDiv' + queryResultId

        $(buttonId).click(function () {
            if ($(this).text() == 'Expand Rows') {
                $(collapseDiv).removeClass('collapseDiv')
                $(this).text('Collapse Rows')
            } else {
                $(collapseDiv).addClass('collapseDiv')
                $(this).text('Expand Rows')
            }
        }).toggle($(collapseDiv).height() >= 300)
    }

    var tableCreateSimpleHead = function (headers, sql, queryResultId) {
        var table = $.tableCreateSimpleHeadHtml(headers, sql, queryResultId)
        $(table).prependTo($('.result'))
    }

    $.tableCreate = function (result, sql, oldResultId, tid, tname) {
        var rowUpdateReady = result.TableName && result.TableName != ""

        var newResultId = ++queryResultId
        var contextMenuHolder = {}
        var table = $.createResultTableHtml(result, sql, rowUpdateReady, newResultId, contextMenuHolder, tid, tname)
        if (oldResultId && oldResultId > 0) {
            $('#executionResultDiv' + oldResultId).replaceWith(table)
        } else {
            $(table).prependTo($('.result'))
        }

        $('#queryResult' + newResultId + ' tbody tr:odd').addClass('rowOdd').attr('rowOdd', 'true')
        $.attachSearchTableEvent(newResultId, 1)
        attachExpandRowsEvent(newResultId)
        attachOpsResultDivEvent(newResultId)
        $.createLinkToTableContextMenu(contextMenuHolder, tid, tname)

        if (rowUpdateReady) {
            $.attachEditableEvent(newResultId)
            attachCopyRowsEvent(newResultId)
            attachDeleteRowsEvent(newResultId)
            $.attachRowTransposesEvent(newResultId)
            $.attachSaveUpdatesEvent(result, newResultId)
        }
    }
})()
