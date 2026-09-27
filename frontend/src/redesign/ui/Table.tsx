import type { ReactNode } from "react";
import { useRedesignI18n } from "../hooks/useRedesignI18n";
import { tableColumns, type TableId } from "../model/tableColumns";
import { EmptyState } from "./StateViews";

export type TableColumnKey<Id extends TableId> = (typeof tableColumns)[Id][number]["key"];

interface TableProps<Id extends TableId, Row> {
  caption?: string;
  empty?: ReactNode;
  renderCell: (row: Row, columnKey: TableColumnKey<Id>) => ReactNode;
  rowKey: (row: Row) => string;
  rows: readonly Row[];
  // Columns come only from tableColumns.ts, so widths always sum to 100%.
  tableId: Id;
}

export function Table<Id extends TableId, Row>({ caption, empty, renderCell, rowKey, rows, tableId }: TableProps<Id, Row>) {
  const { t } = useRedesignI18n();
  const columns = tableColumns[tableId];
  return (
    <div className="tbl-wrap">
      <table className="tbl">
        {caption ? <caption className="visually-hidden">{caption}</caption> : null}
        <colgroup>
          {columns.map((column) => (
            <col key={column.key} style={{ width: `${column.width}%` }} />
          ))}
        </colgroup>
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={column.key} scope="col" title={t(column.labelKey)}>
                {t(column.labelKey)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr className="tbl-empty-row">
              <td className="tbl-empty" colSpan={columns.length}>
                {empty ?? <EmptyState title={t("rd.common.empty")} />}
              </td>
            </tr>
          ) : (
            rows.map((row) => (
              <tr key={rowKey(row)}>
                {columns.map((column) => (
                  <td key={column.key}>{renderCell(row, column.key as TableColumnKey<Id>)}</td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
