from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ComposerPreviewContact")



@_attrs_define
class ComposerPreviewContact:
    """ 
        Attributes:
            email (str):
            first_name (str):
            label (str):
            language (str):
            last_name (str):
            name (str):
            id (str | Unset):
     """

    email: str
    first_name: str
    label: str
    language: str
    last_name: str
    name: str
    id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        first_name = self.first_name

        label = self.label

        language = self.language

        last_name = self.last_name

        name = self.name

        id = self.id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "firstName": first_name,
            "label": label,
            "language": language,
            "lastName": last_name,
            "name": name,
        })
        if id is not UNSET:
            field_dict["id"] = id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        first_name = d.pop("firstName")

        label = d.pop("label")

        language = d.pop("language")

        last_name = d.pop("lastName")

        name = d.pop("name")

        id = d.pop("id", UNSET)

        composer_preview_contact = cls(
            email=email,
            first_name=first_name,
            label=label,
            language=language,
            last_name=last_name,
            name=name,
            id=id,
        )

        return composer_preview_contact

