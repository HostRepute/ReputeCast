from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_thresholds_scope import ReadinessThresholdsScope






T = TypeVar("T", bound="ReadinessThresholds")



@_attrs_define
class ReadinessThresholds:
    """ 
        Attributes:
            bounce (float):
            complaint (float):
            mailbox_complaint (float):
            provider_errors (float):
            scope (ReadinessThresholdsScope):
     """

    bounce: float
    complaint: float
    mailbox_complaint: float
    provider_errors: float
    scope: ReadinessThresholdsScope





    def to_dict(self) -> dict[str, Any]:
        bounce = self.bounce

        complaint = self.complaint

        mailbox_complaint = self.mailbox_complaint

        provider_errors = self.provider_errors

        scope = self.scope.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounce": bounce,
            "complaint": complaint,
            "mailboxComplaint": mailbox_complaint,
            "providerErrors": provider_errors,
            "scope": scope,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        mailbox_complaint = d.pop("mailboxComplaint")

        provider_errors = d.pop("providerErrors")

        scope = ReadinessThresholdsScope(d.pop("scope"))




        readiness_thresholds = cls(
            bounce=bounce,
            complaint=complaint,
            mailbox_complaint=mailbox_complaint,
            provider_errors=provider_errors,
            scope=scope,
        )

        return readiness_thresholds

