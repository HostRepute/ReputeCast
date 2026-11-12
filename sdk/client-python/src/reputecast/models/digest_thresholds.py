from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_thresholds_scope import DigestThresholdsScope






T = TypeVar("T", bound="DigestThresholds")



@_attrs_define
class DigestThresholds:
    """ 
        Attributes:
            bounce (float):
            complaint (float):
            custom (bool):
            scope (DigestThresholdsScope):
     """

    bounce: float
    complaint: float
    custom: bool
    scope: DigestThresholdsScope





    def to_dict(self) -> dict[str, Any]:
        bounce = self.bounce

        complaint = self.complaint

        custom = self.custom

        scope = self.scope.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounce": bounce,
            "complaint": complaint,
            "custom": custom,
            "scope": scope,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        custom = d.pop("custom")

        scope = DigestThresholdsScope(d.pop("scope"))




        digest_thresholds = cls(
            bounce=bounce,
            complaint=complaint,
            custom=custom,
            scope=scope,
        )

        return digest_thresholds

