from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationEmailUpdateBody")



@_attrs_define
class AutomationEmailUpdateBody:
    """ 
        Attributes:
            html (str):
            preheader (str):
            subject (str):
            text (str | Unset):
     """

    html: str
    preheader: str
    subject: str
    text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        html = self.html

        preheader = self.preheader

        subject = self.subject

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "html": html,
            "preheader": preheader,
            "subject": subject,
        })
        if text is not UNSET:
            field_dict["text"] = text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        html = d.pop("html")

        preheader = d.pop("preheader")

        subject = d.pop("subject")

        text = d.pop("text", UNSET)

        automation_email_update_body = cls(
            html=html,
            preheader=preheader,
            subject=subject,
            text=text,
        )

        return automation_email_update_body

