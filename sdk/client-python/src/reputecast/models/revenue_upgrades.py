from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.revenue_move import RevenueMove
  from ..models.revenue_upgrade_row import RevenueUpgradeRow





T = TypeVar("T", bound="RevenueUpgrades")



@_attrs_define
class RevenueUpgrades:
    """ 
        Attributes:
            rows (list[RevenueUpgradeRow]):
            total (RevenueMove):
            window (int):
     """

    rows: list[RevenueUpgradeRow]
    total: RevenueMove
    window: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_move import RevenueMove # noqa: PLC0415
        from ..models.revenue_upgrade_row import RevenueUpgradeRow # noqa: PLC0415
        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        total = self.total.to_dict()

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "rows": rows,
            "total": total,
            "window": window,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_move import RevenueMove # noqa: PLC0415
        from ..models.revenue_upgrade_row import RevenueUpgradeRow # noqa: PLC0415
        d = dict(src_dict)
        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = RevenueUpgradeRow.from_dict(rows_item_data)



            rows.append(rows_item)


        total = RevenueMove.from_dict(d.pop("total"))




        window = d.pop("window")

        revenue_upgrades = cls(
            rows=rows,
            total=total,
            window=window,
        )

        return revenue_upgrades

