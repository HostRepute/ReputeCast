from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.poll_comment import PollComment
  from ..models.poll_comment_counts import PollCommentCounts





T = TypeVar("T", bound="PollCommentList")



@_attrs_define
class PollCommentList:
    """ 
        Attributes:
            counts (PollCommentCounts):
            items (list[PollComment]):
            names (bool):
            total (int):
     """

    counts: PollCommentCounts
    items: list[PollComment]
    names: bool
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.poll_comment import PollComment # noqa: PLC0415
        from ..models.poll_comment_counts import PollCommentCounts # noqa: PLC0415
        counts = self.counts.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        names = self.names

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counts": counts,
            "items": items,
            "names": names,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.poll_comment import PollComment # noqa: PLC0415
        from ..models.poll_comment_counts import PollCommentCounts # noqa: PLC0415
        d = dict(src_dict)
        counts = PollCommentCounts.from_dict(d.pop("counts"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = PollComment.from_dict(items_item_data)



            items.append(items_item)


        names = d.pop("names")

        total = d.pop("total")

        poll_comment_list = cls(
            counts=counts,
            items=items,
            names=names,
            total=total,
        )

        return poll_comment_list

