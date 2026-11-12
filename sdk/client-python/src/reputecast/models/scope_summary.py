from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.scoped_list import ScopedList





T = TypeVar("T", bound="ScopeSummary")



@_attrs_define
class ScopeSummary:
    """ 
        Attributes:
            all_ (bool):
            label (str):
            lists (list[ScopedList]):
            sender_domains (list[str]):
     """

    all_: bool
    label: str
    lists: list[ScopedList]
    sender_domains: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoped_list import ScopedList # noqa: PLC0415
        all_ = self.all_

        label = self.label

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        sender_domains = self.sender_domains




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "label": label,
            "lists": lists,
            "senderDomains": sender_domains,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoped_list import ScopedList # noqa: PLC0415
        d = dict(src_dict)
        all_ = d.pop("all")

        label = d.pop("label")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ScopedList.from_dict(lists_item_data)



            lists.append(lists_item)


        sender_domains = cast(list[str], d.pop("senderDomains"))


        scope_summary = cls(
            all_=all_,
            label=label,
            lists=lists,
            sender_domains=sender_domains,
        )

        return scope_summary

