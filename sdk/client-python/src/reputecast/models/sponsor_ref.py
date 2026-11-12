from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SponsorRef")



@_attrs_define
class SponsorRef:
    """ 
        Attributes:
            id (str):
            initials (str):
            name (str):
            logo_url (str | Unset):
     """

    id: str
    initials: str
    name: str
    logo_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        initials = self.initials

        name = self.name

        logo_url = self.logo_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "initials": initials,
            "name": name,
        })
        if logo_url is not UNSET:
            field_dict["logoUrl"] = logo_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        initials = d.pop("initials")

        name = d.pop("name")

        logo_url = d.pop("logoUrl", UNSET)

        sponsor_ref = cls(
            id=id,
            initials=initials,
            name=name,
            logo_url=logo_url,
        )

        return sponsor_ref

