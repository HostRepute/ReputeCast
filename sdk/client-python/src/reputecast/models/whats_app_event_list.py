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
  from ..models.whats_app_event import WhatsAppEvent





T = TypeVar("T", bound="WhatsAppEventList")



@_attrs_define
class WhatsAppEventList:
    """ 
        Attributes:
            items (list[WhatsAppEvent]):
            next_ (datetime.datetime | Unset):
     """

    items: list[WhatsAppEvent]
    next_: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_event import WhatsAppEvent # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        next_: str | Unset = UNSET
        if not isinstance(self.next_, Unset):
            next_ = self.next_.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
        })
        if next_ is not UNSET:
            field_dict["next"] = next_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_event import WhatsAppEvent # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = WhatsAppEvent.from_dict(items_item_data)



            items.append(items_item)


        _next_ = d.pop("next", UNSET)
        next_: datetime.datetime | Unset
        if isinstance(_next_,  Unset):
            next_ = UNSET
        else:
            next_ = datetime.datetime.fromisoformat(_next_)




        whats_app_event_list = cls(
            items=items,
            next_=next_,
        )

        return whats_app_event_list

