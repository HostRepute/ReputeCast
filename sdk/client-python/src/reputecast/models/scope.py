from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="Scope")



@_attrs_define
class Scope:
    """ 
        Attributes:
            lists (list[str] | Unset):
            sender_domains (list[str] | Unset):
     """

    lists: list[str] | Unset = UNSET
    sender_domains: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        lists: list[str] | Unset = UNSET
        if not isinstance(self.lists, Unset):
            lists = self.lists



        sender_domains: list[str] | Unset = UNSET
        if not isinstance(self.sender_domains, Unset):
            sender_domains = self.sender_domains




        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if lists is not UNSET:
            field_dict["lists"] = lists
        if sender_domains is not UNSET:
            field_dict["senderDomains"] = sender_domains

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        lists = cast(list[str], d.pop("lists", UNSET))


        sender_domains = cast(list[str], d.pop("senderDomains", UNSET))


        scope = cls(
            lists=lists,
            sender_domains=sender_domains,
        )

        return scope

