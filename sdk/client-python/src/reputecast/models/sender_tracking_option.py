from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SenderTrackingOption")



@_attrs_define
class SenderTrackingOption:
    """ 
        Attributes:
            domain (str):
            verified (bool):
     """

    domain: str
    verified: bool





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        verified = self.verified


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "verified": verified,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        verified = d.pop("verified")

        sender_tracking_option = cls(
            domain=domain,
            verified=verified,
        )

        return sender_tracking_option

