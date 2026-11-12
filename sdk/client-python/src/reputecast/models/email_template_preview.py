from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EmailTemplatePreview")



@_attrs_define
class EmailTemplatePreview:
    """ 
        Attributes:
            html (str): The compiled email with its fallbacks shown and its links, scripts and form controls taken out: a
                picture of the email, not a page to use
     """

    html: str





    def to_dict(self) -> dict[str, Any]:
        html = self.html


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "html": html,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        html = d.pop("html")

        email_template_preview = cls(
            html=html,
        )

        return email_template_preview

