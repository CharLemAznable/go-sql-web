(function () {

    function findFileName(resultId, result, cells, columnName) {
        var table = $('#queryResult' + resultId)
        var headRow = table.find('tr.headRow').first().find('td')

        var referFileName = "xxx"
        cells.each(function (jndex, cell) {
            if (jndex > 0) {
                var fieldName = $(headRow.get(jndex + 1)).text()
                if (fieldName !== columnName && fieldName.toLowerCase().indexOf("name") >= 0) {
                    referFileName = $.cellNewValue($(cell))
                    return false
                }
            }
        })

        return referFileName
    }

    $.downloadColumn = function (classifier, tid, tableName, columnName, resultId, result, $cell) {
        var cells = $cell.parent('tr').find('td.dataCell')
        let sql = "select " + columnName + ' from  ' + tableName
            + createWherePart4Download(resultId, result, cells, columnName)
        if ($.currentDriverName === "mysql") sql = sql + ' limit 1'

        var fileNameMaybe = findFileName(resultId, result, cells, columnName)
        let fileName = window.prompt("please input download file name", fileNameMaybe)
        if (fileName == null) return

        $.ajax({
            type: 'POST', url: contextPath + "/downloadColumn",
            data: {tid: tid, sql: sql, fileName: fileName},
            dataType: 'native',
            xhrFields: {
                responseType: 'blob'
            },
            success: function (blob) {
                let link = document.createElement('a')
                link.href = window.URL.createObjectURL(blob)
                link.download = fileName
                link.click()
            },
            error: function (jqXHR, textStatus, errorThrown) {
                $.alertMe(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
            }
        })
    }

    function createWherePart4Download(resultId, result, cells, columnName) {
        var table = $('#queryResult' + resultId)
        var headRow = table.find('tr.headRow').first().find('td')

        if (result.PrimaryKeysIndex.length > 0) {
            var sql = ' where '
            for (var i = 0; i < result.PrimaryKeysIndex.length; ++i) {
                var ki = result.PrimaryKeysIndex[i] + 1
                sql += i > 0 ? ' and ' : ''

                var pkName = $(headRow.get(ki + 1)).text()
                var $cell = $(cells.get(ki))
                let oldValue = $.cellOldValue($cell);
                let dataType = $.cellDataType($cell);
                sql += $.wrapWhereCondition(pkName, oldValue, dataType)
            }
            return sql
        }

        var wherePart = ''
        cells.each(function (jndex, cell) {
            if (jndex > 0) {
                wherePart += wherePart !== '' ? ' and ' : ''

                var fieldName = $(headRow.get(jndex + 1)).text()
                if (fieldName !== columnName) {
                    let $cell = $(cell);
                    let whereValue = $.cellOldValue($cell);
                    let dataType = $.cellDataType($cell);
                    wherePart += $.wrapWhereCondition(fieldName, whereValue, dataType)
                }
            }
        })

        return ' where ' + wherePart
    }

    $.wrapWhereCondition = function (fieldName, fieldValue, dataType) {
        var condition = $.wrapFieldName(fieldName)
        if ("(null)" === fieldValue) {
            condition += ' is null'
        } else if ($.currentDriverName === "goracle") {
            if ("" === fieldValue) {
                condition += ' is null'
            } else if (dataType === "DATE") {
                condition += ' = to_date(\'' + $.formatOracleDateTimeValue(
                    fieldValue, 'YYYY-MM-DD HH:mm:ss')+ '\',\'YYYY-MM-DD HH24:MI:SS\')'
            } else if (dataType.startsWith("TIMESTAMP")) {
                condition += ' = to_timestamp(\'' + $.formatOracleDateTimeValue(
                    fieldValue, 'YYYY-MM-DD HH:mm:ss.SSS') + '\',\'YYYY-MM-DD HH24:MI:SS.FF3\')'
            } else {
                condition += ' = \'' + $.escapeSqlValue(fieldValue) + '\''
            }
        } else {
            condition += ' = \'' + $.escapeSqlValue(fieldValue) + '\''
        }
        return condition
    }
})()