from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.growth_referrer_row import GrowthReferrerRow
  from ..models.growth_rest import GrowthRest





T = TypeVar("T", bound="GrowthReferrers")



@_attrs_define
class GrowthReferrers:
    """ 
        Attributes:
            joins (int):
            none (int):
            other (GrowthRest):
            rows (list[GrowthReferrerRow]):
     """

    joins: int
    none: int
    other: GrowthRest
    rows: list[GrowthReferrerRow]





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_referrer_row import GrowthReferrerRow # noqa: PLC0415
        from ..models.growth_rest import GrowthRest # noqa: PLC0415
        joins = self.joins

        none = self.none

        other = self.other.to_dict()

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "joins": joins,
            "none": none,
            "other": other,
            "rows": rows,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_referrer_row import GrowthReferrerRow # noqa: PLC0415
        from ..models.growth_rest import GrowthRest # noqa: PLC0415
        d = dict(src_dict)
        joins = d.pop("joins")

        none = d.pop("none")

        other = GrowthRest.from_dict(d.pop("other"))




        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = GrowthReferrerRow.from_dict(rows_item_data)



            rows.append(rows_item)


        growth_referrers = cls(
            joins=joins,
            none=none,
            other=other,
            rows=rows,
        )

        return growth_referrers

