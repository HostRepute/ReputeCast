from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ArchivePage")



@_attrs_define
class ArchivePage:
    """ 
        Attributes:
            enabled (bool):
            index (bool):
            lists (list[str]):
            subscribe_box (bool):
     """

    enabled: bool
    index: bool
    lists: list[str]
    subscribe_box: bool





    def to_dict(self) -> dict[str, Any]:
        enabled = self.enabled

        index = self.index

        lists = self.lists



        subscribe_box = self.subscribe_box


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "index": index,
            "lists": lists,
            "subscribeBox": subscribe_box,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        enabled = d.pop("enabled")

        index = d.pop("index")

        lists = cast(list[str], d.pop("lists"))


        subscribe_box = d.pop("subscribeBox")

        archive_page = cls(
            enabled=enabled,
            index=index,
            lists=lists,
            subscribe_box=subscribe_box,
        )

        return archive_page

