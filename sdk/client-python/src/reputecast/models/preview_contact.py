from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from uuid import UUID






T = TypeVar("T", bound="PreviewContact")



@_attrs_define
class PreviewContact:
    """ 
        Attributes:
            email (str):
            gets (str):
            id (UUID):
            language (str):
            name (str):
            offered (bool):
     """

    email: str
    gets: str
    id: UUID
    language: str
    name: str
    offered: bool





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        gets = self.gets

        id = str(self.id)

        language = self.language

        name = self.name

        offered = self.offered


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "gets": gets,
            "id": id,
            "language": language,
            "name": name,
            "offered": offered,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        gets = d.pop("gets")

        id = UUID(d.pop("id"))




        language = d.pop("language")

        name = d.pop("name")

        offered = d.pop("offered")

        preview_contact = cls(
            email=email,
            gets=gets,
            id=id,
            language=language,
            name=name,
            offered=offered,
        )

        return preview_contact

