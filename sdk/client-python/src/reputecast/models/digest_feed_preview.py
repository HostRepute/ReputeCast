from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.digest_feed_item import DigestFeedItem
  from ..models.digest_sent_count import DigestSentCount





T = TypeVar("T", bound="DigestFeedPreview")



@_attrs_define
class DigestFeedPreview:
    """ 
        Attributes:
            items (list[DigestFeedItem]):
            older_before (int):
            older_left_out (int):
            older_new (int):
            older_sent (list[DigestSentCount]):
            total (int):
     """

    items: list[DigestFeedItem]
    older_before: int
    older_left_out: int
    older_new: int
    older_sent: list[DigestSentCount]
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        from ..models.digest_sent_count import DigestSentCount # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        older_before = self.older_before

        older_left_out = self.older_left_out

        older_new = self.older_new

        older_sent = []
        for older_sent_item_data in self.older_sent:
            older_sent_item = older_sent_item_data.to_dict()
            older_sent.append(older_sent_item)



        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "olderBefore": older_before,
            "olderLeftOut": older_left_out,
            "olderNew": older_new,
            "olderSent": older_sent,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_item import DigestFeedItem # noqa: PLC0415
        from ..models.digest_sent_count import DigestSentCount # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = DigestFeedItem.from_dict(items_item_data)



            items.append(items_item)


        older_before = d.pop("olderBefore")

        older_left_out = d.pop("olderLeftOut")

        older_new = d.pop("olderNew")

        older_sent = []
        _older_sent = d.pop("olderSent")
        for older_sent_item_data in (_older_sent):
            older_sent_item = DigestSentCount.from_dict(older_sent_item_data)



            older_sent.append(older_sent_item)


        total = d.pop("total")

        digest_feed_preview = cls(
            items=items,
            older_before=older_before,
            older_left_out=older_left_out,
            older_new=older_new,
            older_sent=older_sent,
            total=total,
        )

        return digest_feed_preview

