from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.campaign_calendar_item import CampaignCalendarItem





T = TypeVar("T", bound="CampaignCalendar")



@_attrs_define
class CampaignCalendar:
    """ 
        Attributes:
            from_ (datetime.datetime):
            items (list[CampaignCalendarItem]):
            to (datetime.datetime):
     """

    from_: datetime.datetime
    items: list[CampaignCalendarItem]
    to: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        from ..models.campaign_calendar_item import CampaignCalendarItem # noqa: PLC0415
        from_ = self.from_.isoformat()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        to = self.to.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "items": items,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.campaign_calendar_item import CampaignCalendarItem # noqa: PLC0415
        d = dict(src_dict)
        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = CampaignCalendarItem.from_dict(items_item_data)



            items.append(items_item)


        to = datetime.datetime.fromisoformat(d.pop("to"))




        campaign_calendar = cls(
            from_=from_,
            items=items,
            to=to,
        )

        return campaign_calendar

