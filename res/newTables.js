tableApp = new Vue({
    el: '#rightTable',
    data() {
        return {
            message: 'Hello Vue!',
            tableNames: [],
            isShow: false,
            tableNameFilterText: '',
            selectMode: false
        }
    },
    computed: {
        switchText() {
            return this.isShow ? '合' : '开'
        },
        showTableNames() {
            let parts = this.tableNameFilterText.split('|')
                .map(item => item.trim())
                .filter(item => item.length > 0)
                .map(item => item.toUpperCase())
            if (parts.length === 0) return this.tableNames;

            return this.tableNames.filter(tableName => parts.some(part => tableName.toUpperCase().indexOf(part) > -1))
        }
    },
    methods: {
        initTable() {
            var self = this
            var tid = activeMerchantId
            var withColumns = !$.withColumnsCache[tid]
            $.ajax({
                type: 'POST',
                url: contextPath + "/query",
                data: {tid: tid, sql: 'initTable', withColumns: withColumns},
                success: function (content, textStatus, request) {
                    if (content && content.Error) {
                        $.alertMe(content.Error)
                        return
                    }

                    if (withColumns) {
                        $.withColumnsCache[tid] = content.TableColumns
                    }

                    $.currentDriverName = content.DriverName
                    if ($.currentDriverName === "mysql") {
                        $.wrapQuoteMark = '`'
                    } else {
                        $.wrapQuoteMark = '"'
                    }

                    self.showTables(content, $.createTableColumns(tid))
                },
                error: function (jqXHR, textStatus, errorThrown) {
                    $.alertMe(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
                }
            })
        },
        reInitTable() {
            var tid = activeMerchantId
            delete $.withColumnsCache[tid]
            tableApp.initTable()
        },
        showTables(content, tableColumns) {
            this.tableNames = content.Rows.map(row => row[1])
            this.isShow = true
            $('.MainDivs').addClass('MainDivsShowTable')

            var timeout = null
            var editor = $.sqlCodeMirror
            editor.off("keyup")
            editor.on("keyup", function (cm, event) {
                if ($.ExcludedIntelliSenseTriggerKeys[(event.keyCode || event.which).toString()]
                    || cm.state.completionActive) return

                var cur = cm.getCursor()
                var ch = cm.getRange(CodeMirror.Pos(cur.line, cur.ch - 1), cur)
                if (!$.isCharForShowHint(ch)) return

                var tok = cm.getTokenAt(cur)
                if (tok.type === "string" && tok.string.length >= 1 && tok.string.substr(0, 1) === "'") return false;

                if (timeout) clearTimeout(timeout)
                timeout = setTimeout(function () {
                    CodeMirror.showHint(cm, CodeMirror.hint.sql, {
                        // "completeSingle: false" prevents case when you are typing some word
                        // and in the middle it is automatically completed and you continue typing by reflex.
                        // So user will always need to select the intended string
                        // from popup (even if it's single option). (copy from @Oleksandr Pshenychnyy)
                        completeSingle: false,
                        tables: tableColumns
                    })
                }, 150)
            })
            $.contextMenu({
                zIndex: 10,
                selector: '.itemSpan',
                callback: function (key, options) {
                    var tableName = $(this).text()

                    if (key === 'ShowFullColumns') {
                        $.executeQueryAjax(activeClassifier, activeMerchantId, activeMerchantCode, activeMerchantName, 'processShowColumn ' + tableName)
                    } else if (key === 'ShowCreateTable') {
                        $.showSqlAjax('showCreateTable ' + tableName)
                    } else if (key === 'RenameTable') {
                        $.appendSqlToSqlEditor('RENAME TABLE ' + tableName + ' TO ' + tableName + "_new", true, false)
                    }
                },
                items: {
                    ShowFullColumns: {name: 'Show Columns', icon: 'columns'},
                    ShowCreateTable: {name: 'Show Create Table', icon: 'create-table'},
                    RenameTable: {name: 'Rename Table', icon: 'create-table'},
                }
            })
        },
        switchShow() {
            if (this.isShow) {
                $('.MainDivs').removeClass('MainDivsShowTable')
            } else {
                $('.MainDivs').addClass('MainDivsShowTable')
            }
            this.isShow = !this.isShow
        }
        ,
        selectTable(tableName) {
            if (this.selectMode) return

            var sql = 'select * from ' + tableName
            var tableQuery = $.SingleTableQuery[tableName.toUpperCase()];

            if (tableQuery) {
                if (tableQuery.replaceSql) sql = tableQuery.replaceSql
                if (tableQuery.appendSql) sql += ' ' + tableQuery.appendSql
            }

            $.executeQueryAjax(activeClassifier, activeMerchantId, activeMerchantCode, activeMerchantName, sql)
        },
        renameTables() {
            const selectedTableName = []
            $("input[type='checkbox'][name='selectedTable']:checked").each((index, item) => {
                selectedTableName.push($(item).attr('value'))
            })
            if (selectedTableName.length <= 0) return alert("No tables checked")

            if ($.currentDriverName === "goracle") {
                $.appendSqlToSqlEditor(selectedTableName.map((table) =>
                    'ALTER TABLE ' + table + ' RENAME TO ' + table + '_{{new}}').join(';\n'), true, false)
                return
            }
            let tables = selectedTableName.map((table) => table + ' to ' + table + '_{{new}}').join(', ')
            $.appendSqlToSqlEditor('RENAME TABLE ' + tables, true, false)
        },
        truncateTables() {
            const selectedTableName = []
            $("input[type='checkbox'][name='selectedTable']:checked").each((index, item) => {
                selectedTableName.push($(item).attr('value'))
            })
            if (selectedTableName.length <= 0) return alert("No tables checked")
            $.executeMultiSqlsAjax(`truncate table ${selectedTableName.join(';\ntruncate table ')};` , true)
        }
    }
})

