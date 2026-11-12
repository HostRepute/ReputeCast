from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="PreviewInput")



@_attrs_define
class PreviewInput:
    """ 
        Attributes:
            contact_id (UUID):
            html (str | Unset):
            mjml (str | Unset):
     """

    contact_id: UUID
    html: str | Unset = UNSET
    mjml: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact_id = str(self.contact_id)

        html = self.html

        mjml = self.mjml


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contactId": contact_id,
        })
        if html is not UNSET:
            field_dict["html"] = html
        if mjml is not UNSET:
            field_dict["mjml"] = mjml

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact_id = UUID(d.pop("contactId"))




        html = d.pop("html", UNSET)

        mjml = d.pop("mjml", UNSET)

        preview_input = cls(
            contact_id=contact_id,
            html=html,
            mjml=mjml,
        )

        return preview_input

