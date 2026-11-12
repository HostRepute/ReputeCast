from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="DomainInput")



@_attrs_define
class DomainInput:
    """ 
        Attributes:
            domain (str): The domain senders use, e.g. news.acme.io
            note (str | Unset): What it is for, e.g. primary sending domain
     """

    domain: str
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
        })
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        note = d.pop("note", UNSET)

        domain_input = cls(
            domain=domain,
            note=note,
        )

        return domain_input

