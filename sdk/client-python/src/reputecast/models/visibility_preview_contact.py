from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VisibilityPreviewContact")



@_attrs_define
class VisibilityPreviewContact:
    """ 
        Attributes:
            email (str):
            id (str):
            language (str):
            name (str):
            trait (str):
            version (int):
     """

    email: str
    id: str
    language: str
    name: str
    trait: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        id = self.id

        language = self.language

        name = self.name

        trait = self.trait

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "id": id,
            "language": language,
            "name": name,
            "trait": trait,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        id = d.pop("id")

        language = d.pop("language")

        name = d.pop("name")

        trait = d.pop("trait")

        version = d.pop("version")

        visibility_preview_contact = cls(
            email=email,
            id=id,
            language=language,
            name=name,
            trait=trait,
            version=version,
        )

        return visibility_preview_contact

