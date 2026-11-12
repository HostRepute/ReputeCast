from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="CatalogLanguage")



@_attrs_define
class CatalogLanguage:
    """ 
        Attributes:
            added (bool):
            builtin_labels (bool):
            code (str):
            contact_codes (list[str]):
            contacts (int):
            name (str):
            native (str):
            regions (list[str]):
            rtl (bool):
     """

    added: bool
    builtin_labels: bool
    code: str
    contact_codes: list[str]
    contacts: int
    name: str
    native: str
    regions: list[str]
    rtl: bool





    def to_dict(self) -> dict[str, Any]:
        added = self.added

        builtin_labels = self.builtin_labels

        code = self.code

        contact_codes = self.contact_codes



        contacts = self.contacts

        name = self.name

        native = self.native

        regions = self.regions



        rtl = self.rtl


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "added": added,
            "builtinLabels": builtin_labels,
            "code": code,
            "contactCodes": contact_codes,
            "contacts": contacts,
            "name": name,
            "native": native,
            "regions": regions,
            "rtl": rtl,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        added = d.pop("added")

        builtin_labels = d.pop("builtinLabels")

        code = d.pop("code")

        contact_codes = cast(list[str], d.pop("contactCodes"))


        contacts = d.pop("contacts")

        name = d.pop("name")

        native = d.pop("native")

        regions = cast(list[str], d.pop("regions"))


        rtl = d.pop("rtl")

        catalog_language = cls(
            added=added,
            builtin_labels=builtin_labels,
            code=code,
            contact_codes=contact_codes,
            contacts=contacts,
            name=name,
            native=native,
            regions=regions,
            rtl=rtl,
        )

        return catalog_language

