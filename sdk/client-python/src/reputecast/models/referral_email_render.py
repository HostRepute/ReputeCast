from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralEmailRender")



@_attrs_define
class ReferralEmailRender:
    """ 
        Attributes:
            for_ (str):
            html (str):
            preheader (str):
            subject (str):
     """

    for_: str
    html: str
    preheader: str
    subject: str





    def to_dict(self) -> dict[str, Any]:
        for_ = self.for_

        html = self.html

        preheader = self.preheader

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "for": for_,
            "html": html,
            "preheader": preheader,
            "subject": subject,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        for_ = d.pop("for")

        html = d.pop("html")

        preheader = d.pop("preheader")

        subject = d.pop("subject")

        referral_email_render = cls(
            for_=for_,
            html=html,
            preheader=preheader,
            subject=subject,
        )

        return referral_email_render

