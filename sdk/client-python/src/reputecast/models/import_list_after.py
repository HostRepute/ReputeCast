from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ImportListAfter")



@_attrs_define
class ImportListAfter:
    """ 
        Attributes:
            contacts (int):
            name (str):
     """

    contacts: int
    name: str





    def to_dict(self) -> dict[str, Any]:
        contacts = self.contacts

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contacts = d.pop("contacts")

        name = d.pop("name")

        import_list_after = cls(
            contacts=contacts,
            name=name,
        )

        return import_list_after

