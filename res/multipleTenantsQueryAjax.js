(function () {
    $.multipleTenantsQueryAjax = function (sql, tenantsMap, multipleExecKeys, resultId, groupIndex, tenantIdsGroup, headerColumnsLen, dataRowsIndex, startTime, batchConfirm) {
        if (groupIndex >= tenantIdsGroup.length || (groupIndex > 0 && batchConfirm && !window.confirm('Continue?'))) {
            $('#queryResult' + resultId + ' tbody tr:odd').addClass('rowOdd').attr('rowOdd', true)
            $.attachSearchTableEvent(resultId)
            $.attachExpandRowsEvent(resultId)
            $.attachOpsResultDivEvent(resultId)

            $('#summaryRows' + resultId).text(dataRowsIndex)
            $('#summaryCostTime' + resultId).text($.costTime(startTime))
            $.createOrderByContextMenu(resultId)
            $.attachHighlightColumnEvent(resultId)
            $.createTableToolsContextMenuInMultiTenantResult(resultId)
            return
        }

        var currentGroup = tenantIdsGroup[groupIndex]
        var multipleTenantIds = currentGroup.join(',')
        $.ajax({
            type: 'POST',
            url: contextPath + "/multipleTenantsQuery",
            data: {multipleTenantIds: multipleTenantIds, sql: sql},
            success: function (content, textStatus, request) {
                if (content && content.length === 1 && content[0].Tid === "" && content[0].Error !== "") {
                    // Maybe :
                    // [{"Headers":null,"Rows":null,"Error":"dangerous sql, please get authorized first!","ExecutionTime":"2018-03-09 13:41:06.443",
                    // "CostTime":"8.591µs","DatabaseName":"","TableName":"","PrimaryKeysIndex":null,"Msg":"","Tid":""}]
                    $.alertMe(content[0].Error)
                    return
                }

                var headerHolder = {}
                var resortedContent = sortContent(content, currentGroup, headerHolder, groupIndex)
                if (groupIndex == 0) {
                    tableCreateSimpleHead(multipleExecKeys, headerHolder.Headers, sql, resultId)
                    headerColumnsLen = headerHolder.Headers ? headerHolder.Headers.length : 1
                }

                var rows = ''
                for (var i = 0; i < resortedContent.length; ++i) {
                    rows += createRowsSimple(tenantsMap, multipleExecKeys, resortedContent[i], headerColumnsLen, dataRowsIndex)
                    dataRowsIndex += resortedContent[i].Rows && resortedContent[i].Rows.length ? resortedContent[i].Rows.length : 1
                }

                $('#queryResult' + resultId + " tbody").append(rows)

                setTimeout(function () { // Leave time for rendering, and then continue to next batch.
                    $.multipleTenantsQueryAjax(sql, tenantsMap, multipleExecKeys, resultId, groupIndex + 1, tenantIdsGroup, headerColumnsLen, dataRowsIndex, startTime, batchConfirm)
                }, 100)
            },
            error: function (jqXHR, textStatus, errorThrown) {
                $.alertMe(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
            }
        })
    }

    var tableCreateSimpleHeadHtml = function (multipleExecKeys, headers, sql, resultId) {
        var seqNum = $.convertSeqNum(resultId)
        var table = '<div class="executionResult" id="executionResultDiv' + resultId + '">' +
            '<table class="executionSummary"><tr>' +
            '<td class="resultId" id="resultId' + resultId + '">#' + seqNum + '</td>' +
            '<td>Tenant:&nbsp;N/A</td><td>Db:&nbsp;N/A</td>' +
            '<td>Table:&nbsp;<span class="tableTools" id="tableTools' + resultId + '">...</span></td>' +
            '<td>Rows:&nbsp;<span id="summaryRows' + resultId + '">0</span></td>' +
            '<td>Time:&nbsp;' + $.js_yyyy_mm_dd_hh_mm_ss_SSS() + '</td>' +
            '<td>Cost:&nbsp;<span id="summaryCostTime' + resultId + '">0</span></td>' +
            '<td><span class="opsSpan" id="screenShot' + resultId + '">截图</span></td>' +
            '<td><span class="opsSpan" id="closeResult' + resultId + '">Close</span></td>' +
            '<td><span class="opsSpan" id="closeOthers' + resultId + '">CloseOthers</span></td>' +
            '</tr>' +
            '</table>'
        table += '<div id="divResult' + resultId + '" class="divResult">'
        table += '<div class="operateAreaDiv">'
        table += '<input id="searchTable' + resultId + '" class="searchTable" placeholder="Type to search">'
        table += '<button id="expandRows' + resultId + '"><span class="context-menu-icons context-menu-icon-expand"></span></button>'
        table += '<span class="sqlTd">' + sql + '</span>'
        table += '</div>'
        table += '<div id="collapseDiv' + resultId + `" class="collapseDiv">`

        table += '<table id="queryResult' + resultId + `" class="queryResult">`
        table += '<thead><tr class="headRow"><td></td>'
        table += '<td class="headCell">#</td>'
        for (var i = 0; i < multipleExecKeys.length; ++i) {
            table += '<td class="headCell">' + multipleExecKeys[i] + '</td>'
        }
        table += '<td class="headCell">##</td>'

        if (headers && headers.length) {
            for (var j = 0; j < headers.length; ++j) {
                table += '<td class="headCell contextMenu ' + $.escapeContextMenuCssName(headers[j]) + '">' + headers[j] + '</td>'
            }
        } else {
            table += '<td class="headCell">Msg</td>'
        }
        table += '</tr></thead>'

        table += '<tbody></tbody></table></div><div></div>'

        return table
    }

    var createRowsSimple = function (tenantsMap, multipleExecKeys, result, headerColumnsLen, dataRowsIndex) {
        var rowHtml = ''
        var tenant = tenantsMap[result.Tid]

        if (result.Rows && result.Rows.length) {
            var totalLen = result.Rows.length
            var beforeLen = parseInt(totalLen / 5) * 5
            var splitLen = totalLen - beforeLen
            for (var i = 0; i < totalLen; i++) {
                rowHtml += '<tr class="dataRow"><td></td>'
                rowHtml += '<td class="dataCell">' + (dataRowsIndex + i + 1) + '</td>'
                if (i % 5 == 0) {
                    var rowspan = i < beforeLen ? 5 : splitLen
                    for (var k = 0; k < multipleExecKeys.length; ++k) {
                        rowHtml += '<td class="dataCell" rowspan="' + rowspan + '">' + tenant[multipleExecKeys[k]] + '</td>'
                    }
                } else {
                    for (var l = 0; l < multipleExecKeys.length; ++l) {
                        rowHtml += '<td class="dataCell rowspanned">' + tenant[multipleExecKeys[l]] + '</td>'
                    }
                }

                rowHtml += '<td>' + (i + 1) + '</td>'

                for (var j = 0; j < result.Rows[i].length; ++j) {
                    var cellValue = result.Rows[i][j]

                    rowHtml += '<td class="dataCell">' + cellValue + '</td>'
                }

                rowHtml += '</tr>'
            }
        } else {
            rowHtml += '<tr class="dataRow"><td></td>'
            rowHtml += '<td class="dataCell">' + (dataRowsIndex + 1) + '</td>'
            for (var m = 0; m < multipleExecKeys.length; ++m) {
                rowHtml += '<td class="dataCell">' + tenant[multipleExecKeys[m]] + '</td>'
            }
            if (result.Error !== "" || result.Msg !== "") {
                rowHtml += '<td>1</td><td class="dataCell ' + (result.Error !== '' ? 'error' : '') + '" ' +
                    'colspan="' + (headerColumnsLen + 1) + '">' + (result.Error || result.Msg) + '</td>'
            } else {
                rowHtml += '<td>1</td><td class="dataCell ' + (result.Error !== '' ? 'error' : '') + '" ' +
                    'colspan="' + (headerColumnsLen + 1) + '">0 rows returned</td>'
            }
            rowHtml += '</tr>'
        }
        return rowHtml
    }


    var tableCreateSimpleHead = function (multipleExecKeys, headers, sql, queryResultId) {
        var table = tableCreateSimpleHeadHtml(multipleExecKeys, headers, sql, queryResultId)
        $(table).prependTo($('.result'))
    }

    $.findTenants = function (resultId, multipleExecKeys, multipleExecIndexes) {
        var chosenRows = $.chosenRowsHighlightedOrAll(resultId)

        var offset = 2
        var tenants = []
        chosenRows.each(function (index, tr) {
            var tds = $(tr).find('td');
            var item = {}
            for (var i = 0; i < multipleExecKeys.length; ++i) {
                var key = multipleExecKeys[i]
                var keyIndex = parseInt(multipleExecIndexes[i])
                item[key] = tds.eq(keyIndex + offset).text()
            }

            tenants.push(item)
        })

        return tenants
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

})()