(function () {
    const regex = new RegExp(/[\n\r']/g);

    $.escapeSqlValue = function (value) {
        return value.replace(regex, function (char) {
            const m = ['\n', '\r', "'"];
            const r = ['\\n', '\\r', "''"];
            return r[m.indexOf(char)]
        })
    };

    $.formatOracleDateTimeValue = function (value, format) {
        if (!value.endsWith("Z")) value += "Z";
        return dayjs(value).utc().format(format);
    }

    function createValuePrefix(result) {
        if ($.currentDriverName === "goracle") {
            return 'into ' + wrapTableName(result.TableName) + '(' + createFieldNamesList(result) + ') values'
        }
        // mysql/pg/go-sqlite3
        return ""
    }

    function createValuePart(cells) {
        let valueSql = '(';
        cells.each(function (index, cell) {
            valueSql += index > 1 ? ', ' : '';
            if (index > 0) {
                let $cell = $(cell);
                let newValue = $.cellNewValue($cell);
                let dataType = $.cellDataType($cell);
                if ("(null)" === newValue) {
                    valueSql += 'null'
                } else if ($.currentDriverName === "goracle" &&
                    (dataType === "DATE" || dataType.startsWith("TIMESTAMP"))) {
                    if (dataType === "DATE") {
                        valueSql += 'to_date(\'' + $.formatOracleDateTimeValue(newValue,
                            'YYYY-MM-DD HH:mm:ss') + '\',\'YYYY-MM-DD HH24:MI:SS\')'
                    } else if (dataType.startsWith("TIMESTAMP")) {
                        valueSql += 'to_timestamp(\'' + $.formatOracleDateTimeValue(newValue,
                            'YYYY-MM-DD HH:mm:ss.SSS') + '\',\'YYYY-MM-DD HH24:MI:SS.FF3\')'
                    }
                } else {
                    valueSql += '\'' + $.escapeSqlValue(newValue) + '\''
                }
            }
        });
        return valueSql + ')'
    }

    function joinValues(values) {
        if ($.currentDriverName === "goracle") {
            return values.join('\n') + "\nselect 1 from dual"
        }
        return values.join(',\n')
    }

    $.createInsert = function (cells, result) {
        if ($.currentDriverName === "goracle") {
            return 'insert ' + createValuePrefix(result) + createValuePart(cells)
        }
        return $.createInsertSqlPrefix(result) + createValuePart(cells)
    }

    function createFieldNamesList(result) {
        var headers = result.Headers
        var fieldNames = ''
        for (var i = 0; i < headers.length; ++i) {
            fieldNames += i > 0 ? ', ' : ''
            fieldNames += wrapFieldName(headers[i])
        }

        return fieldNames
    }

    $.createInsertSqlPrefix = function (result) {
        if ($.currentDriverName === "goracle") {
            return 'insert all '
        }
        return 'insert into ' + wrapTableName(result.TableName) + '(' + createFieldNamesList(result) + ') values'
    }

    $.createSelectEqlTemplate = function (result) {
        return 'select ' + createFieldNamesList(result) + '\nfrom ' + wrapTableName(result.TableName) + '\nwhere ' + createWhereItems(result)
    }
    $.createUpdateEqlTemplate = function (result) {
        return 'update ' + wrapTableName(result.TableName) + '\nset ' + createSetItems(result) + '\nwhere ' + createWhereItems(result)
    }
    $.createDeleteEqlTemplate = function (result) {
        return 'delete from ' + wrapTableName(result.TableName) + '\nwhere ' + createWhereItems(result)
    }

    $.createJavaBean = function (tid, result) {
        var bean = 'import lombok.*;\n' +
            '\n' +
            '@Data @AllArgsConstructor @NoArgsConstructor @Builder\n'

        var tableName = result.TableName || 'xxx'
        bean += 'public class ' + $.CamelCased(tableName) + ' {'

        var tableComment = $.findTableComment(tid, tableName)
        if (tableComment !== "") {
            bean += ' // ' + $.mergeLines(tableComment)
        }

        bean += '\n' + $.createJavaBeanFieldNamesList(tid, tableName)
        bean += '}'

        return bean
    }

    function createSetItems(result) {
        var headers = result.Headers

        var sql = ''
        for (var i = 0; i < headers.length; ++i) {
            sql += sql != '' ? ',\n' : ''
            var fieldName = headers[i]
            sql += wrapFieldName(fieldName) + ' = \'#' + $.camelCased(fieldName) + '#\''
        }

        return sql
    }

    function createWhereItems(result) {
        var pkIndexes = result.PrimaryKeysIndex;
        var headers = result.Headers

        var sql = ''
        if (pkIndexes.length > 0) {
            for (var i = 0; i < pkIndexes.length; ++i) {
                var ki = pkIndexes[i]
                sql += i > 0 ? '\nand ' : ''

                var pkName = headers[ki]
                sql += wrapFieldName(pkName) + ' = \'#' + $.camelCased(pkName) + '#\''
            }
            return sql
        } else {
            var wherePart = ''
            for (var i = 0; i < headers.length; ++i) {
                wherePart += wherePart != '' ? '\nand ' : ''
                var fieldName = headers[i]
                wherePart += wrapFieldName(fieldName) + ' = \'#' + $.camelCased(fieldName) + '#\''
            }
            sql += wherePart
        }

        return sql
    }

    function createWhereClause(result, cells) {
        var headers = result.Headers
        var where = ''
        if (result.PrimaryKeysIndex.length > 0) {
            for (var i = 0; i < result.PrimaryKeysIndex.length; ++i) {
                var ki = result.PrimaryKeysIndex[i]
                where += i > 0 ? ' and ' : ''

                var pkName = headers[ki]
                let $eq = cells.eq(ki + 1);
                let pkValue = $.cellOldValue($eq);
                let dataType = $.cellDataType($eq);
                where += $.wrapWhereCondition(pkName, pkValue, dataType)
            }
        } else {
            var wherePart = ''
            cells.each(function (index, cell) {
                if (index > 0) {
                    wherePart += wherePart != '' ? ' and ' : ''
                    var fieldName = headers[index - 1]

                    let $cell = $(cell);
                    let whereValue = $.cellOldValue($cell);
                    let dataType = $.cellDataType($cell);
                    wherePart += $.wrapWhereCondition(fieldName, whereValue, dataType)
                }
            })
            where += wherePart
        }
        return where;
    }


    $.createInsertEqlTemplate = function (result) {
        var values = 'insert into ' + wrapTableName(result.TableName) + '(' + createFieldNamesList(result) + ')\nvalues('
        var headers = result.Headers
        for (var i = 0; i < headers.length; ++i) {
            values += i > 0 ? ', ' : ''
            values += '\'#' + $.camelCased(headers[i]) + '#\''
        }
        return values + ')'
    }

    $.createSelectSql = function (result) {
        var sql = 'select '

        var headers = result.Headers
        for (var i = 0; i < headers.length; ++i) {
            sql += i > 0 ? ', ' : ''
            sql += wrapFieldName(headers[i])
        }

        return sql + ' from ' + wrapTableName(result.TableName)
    }

    $.createSelectSqls = function (selectSql, result, resultId) {
        var tbody = $('#queryResult' + resultId + ' tbody')
        var values = []
        tbody.find('tr.highlight:visible').each(function (index, tr) {
            var cells = $(tr).find('td.dataCell')
            var valuePart = createSelectForRow(selectSql, result, cells)
            values.push(valuePart)
        })

        return values.join(';\n')
    }


    var createSelectForRow = function (selectSql, result, cells) {
        var sql = selectSql + ' where '
        var where = createWhereClause(result, cells)
        return sql + where
    }

    $.createDeleteSqls = function (result, resultId) {
        var tbody = $('#queryResult' + resultId + ' tbody')
        var values = []
        tbody.find('tr.highlight:visible').each(function (index, tr) {
            var cells = $(tr).find('td.dataCell')
            var valuePart = createDeleteForRow(result, cells)
            values.push(valuePart)
        })

        return values.join(';\n')
    }


    var createDeleteForRow = function (result, cells) {
        var sql = 'delete from ' + wrapTableName(result.TableName) + ' where '
        var where = createWhereClause(result, cells)
        return sql + where
    }


    $.createInsertValuesHighlighted = function (resultId, result) {
        var tbody = $('#queryResult' + resultId + ' tbody')
        var values = []
        var valuePrefix = createValuePrefix(result)
        tbody.find('tr.highlight:visible').each(function (index, tr) {
            var cells = $(tr).find('td.dataCell')
            var valuePart = valuePrefix + createValuePart(cells)
            values.push(valuePart)
        })

        return joinValues(values)
    }

    $.createInsertValuesAll = function (resultId, result) {
        var tbody = $('#queryResult' + resultId + ' tbody')
        var values = []
        var valuePrefix = createValuePrefix(result)
        tbody.find('tr:visible').each(function (index, tr) {
            var cells = $(tr).find('td.dataCell')
            var valuePart = valuePrefix + createValuePart(cells)
            values.push(valuePart)
        })

        return joinValues(values)
    }

    $.createUpdateSetPart = function (cells, result, headRow) {
        var updateSql = null
        cells.each(function (jndex, cell) {
            var changedCell = $(this).hasClass('changedCell')
            if (changedCell) {
                if (updateSql == null) {
                    updateSql = 'update ' + wrapTableName(result.TableName) + ' set '
                } else {
                    updateSql += ', '
                }
                var fieldName = $(headRow.get(jndex + 1)).text()
                let $cell = $(cell);
                let newValue = $.cellNewValue($cell);
                let dataType = $.cellDataType($cell);
                updateSql += $.wrapWhereCondition(fieldName, newValue, dataType)
            }
        })
        return updateSql
    }

    $.cellNewValue = function ($cell) {
        // return $.trim($cell.hasClass('textAreaTd') ? $cell.find('textarea').val() : $cell.text())
        let text = $cell.hasClass('textAreaTd') ? $cell.find('textarea').val() : $cell.text();
        if ($.currentDriverName === "mysql") text = $.trim(text)
        return text
    }

    $.wrapQuoteMark = '`'

    function wrapFieldName(fieldName) {
        if (fieldName.indexOf('_') >= 0) return fieldName
        else return $.wrapQuoteMark + fieldName + $.wrapQuoteMark
    }

    $.wrapFieldName = wrapFieldName

    function wrapTableName(tableName) {
        let tn = tableName || 'xxx'
        if ($.wrapQuoteMark === '`')
            return $.wrapQuoteMark + tn + $.wrapQuoteMark
        else return tn
    }

    $.wrapTableName = wrapTableName

    var cellOldValue = function ($cell) {
        var old = $cell.attr('old');
        return old === undefined ? $cell.text() : old
    }

    $.cellOldValue = cellOldValue

    var cellDataType = function ($cell) {
        return ($cell.attr("sql_data_type") || "").toUpperCase();
    }

    $.cellDataType = cellDataType

    $.createWherePart = function (result, headRow, cells) {
        var sql = ' where '
        if (result.PrimaryKeysIndex.length > 0) {
            for (var i = 0; i < result.PrimaryKeysIndex.length; ++i) {
                var ki = result.PrimaryKeysIndex[i] + 1
                sql += i > 0 ? ' and ' : ''

                var pkName = $(headRow.get(ki + 1)).text()
                let $cell = $(cells.get(ki))
                let fieldValue = $.cellOldValue($cell);
                let dataType = $.cellDataType($cell);
                sql += $.wrapWhereCondition(pkName, fieldValue, dataType)
            }
            return sql
        } else {
            var wherePart = ''
            cells.each(function (jndex, cell) {
                if (jndex > 0) {
                    wherePart += wherePart !== '' ? ' and ' : ''

                    var fieldName = $(headRow.get(jndex + 1)).text()
                    let $this = $(this);
                    let oldValue = $.cellOldValue($this);
                    let dataType = $.cellDataType($this);
                    wherePart += $.wrapWhereCondition(fieldName, oldValue, dataType)
                }
            })

            sql += wherePart
        }

        return sql
    }
})()