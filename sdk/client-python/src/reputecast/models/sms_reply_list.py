from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sms_reply import SmsReply
  from ..models.sms_reply_list_counts import SmsReplyListCounts





T = TypeVar("T", bound="SmsReplyList")



@_attrs_define
class SmsReplyList:
    """ 
        Attributes:
            counts (SmsReplyListCounts):
            items (list[SmsReply]):
     """

    counts: SmsReplyListCounts
    items: list[SmsReply]





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_reply import SmsReply # noqa: PLC0415
        from ..models.sms_reply_list_counts import SmsReplyListCounts # noqa: PLC0415
        counts = self.counts.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counts": counts,
            "items": items,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_reply import SmsReply # noqa: PLC0415
        from ..models.sms_reply_list_counts import SmsReplyListCounts # noqa: PLC0415
        d = dict(src_dict)
        counts = SmsReplyListCounts.from_dict(d.pop("counts"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SmsReply.from_dict(items_item_data)



            items.append(items_item)


        sms_reply_list = cls(
            counts=counts,
            items=items,
        )

        return sms_reply_list

