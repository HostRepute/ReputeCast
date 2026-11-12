from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="AutomationEmailPreview")



@_attrs_define
class AutomationEmailPreview:
    """ 
        Attributes:
            contact (str):
            html (str):
            preheader (str):
            problems (list[str]):
            subject (str):
     """

    contact: str
    html: str
    preheader: str
    problems: list[str]
    subject: str





    def to_dict(self) -> dict[str, Any]:
        contact = self.contact

        html = self.html

        preheader = self.preheader

        problems = self.problems



        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contact": contact,
            "html": html,
            "preheader": preheader,
            "problems": problems,
            "subject": subject,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact = d.pop("contact")

        html = d.pop("html")

        preheader = d.pop("preheader")

        problems = cast(list[str], d.pop("problems"))


        subject = d.pop("subject")

        automation_email_preview = cls(
            contact=contact,
            html=html,
            preheader=preheader,
            problems=problems,
            subject=subject,
        )

        return automation_email_preview

