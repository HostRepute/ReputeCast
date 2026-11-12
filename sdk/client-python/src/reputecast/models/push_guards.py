from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.push_sent_today import PushSentToday
  from ..models.push_zone import PushZone





T = TypeVar("T", bound="PushGuards")



@_attrs_define
class PushGuards:
    """ 
        Attributes:
            cap (int):
            held (int):
            inside (int):
            over_cap (int):
            quiet (bool):
            sent_today (list[PushSentToday]):
            unknown (int):
            window (str):
            zones (list[PushZone]):
     """

    cap: int
    held: int
    inside: int
    over_cap: int
    quiet: bool
    sent_today: list[PushSentToday]
    unknown: int
    window: str
    zones: list[PushZone]





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_sent_today import PushSentToday # noqa: PLC0415
        from ..models.push_zone import PushZone # noqa: PLC0415
        cap = self.cap

        held = self.held

        inside = self.inside

        over_cap = self.over_cap

        quiet = self.quiet

        sent_today = []
        for sent_today_item_data in self.sent_today:
            sent_today_item = sent_today_item_data.to_dict()
            sent_today.append(sent_today_item)



        unknown = self.unknown

        window = self.window

        zones = []
        for zones_item_data in self.zones:
            zones_item = zones_item_data.to_dict()
            zones.append(zones_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cap": cap,
            "held": held,
            "inside": inside,
            "overCap": over_cap,
            "quiet": quiet,
            "sentToday": sent_today,
            "unknown": unknown,
            "window": window,
            "zones": zones,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_sent_today import PushSentToday # noqa: PLC0415
        from ..models.push_zone import PushZone # noqa: PLC0415
        d = dict(src_dict)
        cap = d.pop("cap")

        held = d.pop("held")

        inside = d.pop("inside")

        over_cap = d.pop("overCap")

        quiet = d.pop("quiet")

        sent_today = []
        _sent_today = d.pop("sentToday")
        for sent_today_item_data in (_sent_today):
            sent_today_item = PushSentToday.from_dict(sent_today_item_data)



            sent_today.append(sent_today_item)


        unknown = d.pop("unknown")

        window = d.pop("window")

        zones = []
        _zones = d.pop("zones")
        for zones_item_data in (_zones):
            zones_item = PushZone.from_dict(zones_item_data)



            zones.append(zones_item)


        push_guards = cls(
            cap=cap,
            held=held,
            inside=inside,
            over_cap=over_cap,
            quiet=quiet,
            sent_today=sent_today,
            unknown=unknown,
            window=window,
            zones=zones,
        )

        return push_guards

