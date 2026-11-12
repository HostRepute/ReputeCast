from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProviderLimitsInput")



@_attrs_define
class ProviderLimitsInput:
    """ 
        Attributes:
            bounce (float):
            complaint (float):
     """

    bounce: float
    complaint: float





    def to_dict(self) -> dict[str, Any]:
        bounce = self.bounce

        complaint = self.complaint


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounce": bounce,
            "complaint": complaint,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        provider_limits_input = cls(
            bounce=bounce,
            complaint=complaint,
        )

        return provider_limits_input

