from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.report_bounce_row import ReportBounceRow





T = TypeVar("T", bound="ReportBounceTable")



@_attrs_define
class ReportBounceTable:
    """ 
        Attributes:
            rows (list[ReportBounceRow]):
            total (int):
     """

    rows: list[ReportBounceRow]
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_bounce_row import ReportBounceRow # noqa: PLC0415
        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "rows": rows,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_bounce_row import ReportBounceRow # noqa: PLC0415
        d = dict(src_dict)
        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = ReportBounceRow.from_dict(rows_item_data)



            rows.append(rows_item)


        total = d.pop("total")

        report_bounce_table = cls(
            rows=rows,
            total=total,
        )

        return report_bounce_table

