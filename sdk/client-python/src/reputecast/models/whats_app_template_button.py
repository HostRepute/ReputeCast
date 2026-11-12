from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_template_button_type import WhatsAppTemplateButtonType
from ..types import UNSET, Unset






T = TypeVar("T", bound="WhatsAppTemplateButton")



@_attrs_define
class WhatsAppTemplateButton:
    """ 
        Attributes:
            text (str):
            type_ (WhatsAppTemplateButtonType):
            url (str | Unset):
     """

    text: str
    type_: WhatsAppTemplateButtonType
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        text = self.text

        type_ = self.type_.value

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "text": text,
            "type": type_,
        })
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        text = d.pop("text")

        type_ = WhatsAppTemplateButtonType(d.pop("type"))




        url = d.pop("url", UNSET)

        whats_app_template_button = cls(
            text=text,
            type_=type_,
            url=url,
        )

        return whats_app_template_button

