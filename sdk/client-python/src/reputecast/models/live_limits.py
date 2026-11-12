from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveLimits")



@_attrs_define
class LiveLimits:
    """ 
        Attributes:
            api_errors (float):
            bounce (float):
            complaint (float):
            deferral (float):
            mailbox_complaint (float):
     """

    api_errors: float
    bounce: float
    complaint: float
    deferral: float
    mailbox_complaint: float





    def to_dict(self) -> dict[str, Any]:
        api_errors = self.api_errors

        bounce = self.bounce

        complaint = self.complaint

        deferral = self.deferral

        mailbox_complaint = self.mailbox_complaint


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "apiErrors": api_errors,
            "bounce": bounce,
            "complaint": complaint,
            "deferral": deferral,
            "mailboxComplaint": mailbox_complaint,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_errors = d.pop("apiErrors")

        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        deferral = d.pop("deferral")

        mailbox_complaint = d.pop("mailboxComplaint")

        live_limits = cls(
            api_errors=api_errors,
            bounce=bounce,
            complaint=complaint,
            deferral=deferral,
            mailbox_complaint=mailbox_complaint,
        )

        return live_limits

