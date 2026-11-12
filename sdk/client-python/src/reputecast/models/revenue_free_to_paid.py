from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.revenue_source_row import RevenueSourceRow





T = TypeVar("T", bound="RevenueFreeToPaid")



@_attrs_define
class RevenueFreeToPaid:
    """ 
        Attributes:
            all_ (int):
            direct (int):
            rows (list[RevenueSourceRow]):
            total (RevenueSourceRow):
     """

    all_: int
    direct: int
    rows: list[RevenueSourceRow]
    total: RevenueSourceRow





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_source_row import RevenueSourceRow # noqa: PLC0415
        all_ = self.all_

        direct = self.direct

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        total = self.total.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "direct": direct,
            "rows": rows,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_source_row import RevenueSourceRow # noqa: PLC0415
        d = dict(src_dict)
        all_ = d.pop("all")

        direct = d.pop("direct")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = RevenueSourceRow.from_dict(rows_item_data)



            rows.append(rows_item)


        total = RevenueSourceRow.from_dict(d.pop("total"))




        revenue_free_to_paid = cls(
            all_=all_,
            direct=direct,
            rows=rows,
            total=total,
        )

        return revenue_free_to_paid

