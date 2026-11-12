from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sender_verification_status import SenderVerificationStatus






T = TypeVar("T", bound="SenderVerification")



@_attrs_define
class SenderVerification:
    """ 
        Attributes:
            provider_id (str):
            status (SenderVerificationStatus):
            text (str):
     """

    provider_id: str
    status: SenderVerificationStatus
    text: str





    def to_dict(self) -> dict[str, Any]:
        provider_id = self.provider_id

        status = self.status.value

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "providerId": provider_id,
            "status": status,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        provider_id = d.pop("providerId")

        status = SenderVerificationStatus(d.pop("status"))




        text = d.pop("text")

        sender_verification = cls(
            provider_id=provider_id,
            status=status,
            text=text,
        )

        return sender_verification

