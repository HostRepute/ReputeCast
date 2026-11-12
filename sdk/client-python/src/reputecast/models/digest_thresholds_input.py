from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_thresholds_input_scope import DigestThresholdsInputScope






T = TypeVar("T", bound="DigestThresholdsInput")



@_attrs_define
class DigestThresholdsInput:
    """ 
        Attributes:
            bounce (float):
            complaint (float):
            scope (DigestThresholdsInputScope):
     """

    bounce: float
    complaint: float
    scope: DigestThresholdsInputScope





    def to_dict(self) -> dict[str, Any]:
        bounce = self.bounce

        complaint = self.complaint

        scope = self.scope.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounce": bounce,
            "complaint": complaint,
            "scope": scope,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        scope = DigestThresholdsInputScope(d.pop("scope"))




        digest_thresholds_input = cls(
            bounce=bounce,
            complaint=complaint,
            scope=scope,
        )

        return digest_thresholds_input

