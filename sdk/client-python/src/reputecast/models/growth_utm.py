from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.growth_rest import GrowthRest
  from ..models.growth_utm_row import GrowthUTMRow





T = TypeVar("T", bound="GrowthUTM")



@_attrs_define
class GrowthUTM:
    """ 
        Attributes:
            captured (int):
            joins (int):
            none (GrowthUTMRow):
            not_captured (int):
            other (GrowthRest):
            rows (list[GrowthUTMRow]):
            capture_started_at (datetime.datetime | Unset):
            not_captured_human_open_rate (float | Unset):
     """

    captured: int
    joins: int
    none: GrowthUTMRow
    not_captured: int
    other: GrowthRest
    rows: list[GrowthUTMRow]
    capture_started_at: datetime.datetime | Unset = UNSET
    not_captured_human_open_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_rest import GrowthRest # noqa: PLC0415
        from ..models.growth_utm_row import GrowthUTMRow # noqa: PLC0415
        captured = self.captured

        joins = self.joins

        none = self.none.to_dict()

        not_captured = self.not_captured

        other = self.other.to_dict()

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        capture_started_at: str | Unset = UNSET
        if not isinstance(self.capture_started_at, Unset):
            capture_started_at = self.capture_started_at.isoformat()

        not_captured_human_open_rate = self.not_captured_human_open_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "captured": captured,
            "joins": joins,
            "none": none,
            "notCaptured": not_captured,
            "other": other,
            "rows": rows,
        })
        if capture_started_at is not UNSET:
            field_dict["captureStartedAt"] = capture_started_at
        if not_captured_human_open_rate is not UNSET:
            field_dict["notCapturedHumanOpenRate"] = not_captured_human_open_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_rest import GrowthRest # noqa: PLC0415
        from ..models.growth_utm_row import GrowthUTMRow # noqa: PLC0415
        d = dict(src_dict)
        captured = d.pop("captured")

        joins = d.pop("joins")

        none = GrowthUTMRow.from_dict(d.pop("none"))




        not_captured = d.pop("notCaptured")

        other = GrowthRest.from_dict(d.pop("other"))




        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = GrowthUTMRow.from_dict(rows_item_data)



            rows.append(rows_item)


        _capture_started_at = d.pop("captureStartedAt", UNSET)
        capture_started_at: datetime.datetime | Unset
        if isinstance(_capture_started_at,  Unset):
            capture_started_at = UNSET
        else:
            capture_started_at = datetime.datetime.fromisoformat(_capture_started_at)




        not_captured_human_open_rate = d.pop("notCapturedHumanOpenRate", UNSET)

        growth_utm = cls(
            captured=captured,
            joins=joins,
            none=none,
            not_captured=not_captured,
            other=other,
            rows=rows,
            capture_started_at=capture_started_at,
            not_captured_human_open_rate=not_captured_human_open_rate,
        )

        return growth_utm

