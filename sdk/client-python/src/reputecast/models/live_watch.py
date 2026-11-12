from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.live_threshold import LiveThreshold





T = TypeVar("T", bound="LiveWatch")



@_attrs_define
class LiveWatch:
    """ 
        Attributes:
            active (bool):
            alert (bool):
            hours (int):
            limit (float):
            rows (list[LiveThreshold]):
            until (datetime.datetime):
     """

    active: bool
    alert: bool
    hours: int
    limit: float
    rows: list[LiveThreshold]
    until: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_threshold import LiveThreshold # noqa: PLC0415
        active = self.active

        alert = self.alert

        hours = self.hours

        limit = self.limit

        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "alert": alert,
            "hours": hours,
            "limit": limit,
            "rows": rows,
            "until": until,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_threshold import LiveThreshold # noqa: PLC0415
        d = dict(src_dict)
        active = d.pop("active")

        alert = d.pop("alert")

        hours = d.pop("hours")

        limit = d.pop("limit")

        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = LiveThreshold.from_dict(rows_item_data)



            rows.append(rows_item)


        until = datetime.datetime.fromisoformat(d.pop("until"))




        live_watch = cls(
            active=active,
            alert=alert,
            hours=hours,
            limit=limit,
            rows=rows,
            until=until,
        )

        return live_watch

