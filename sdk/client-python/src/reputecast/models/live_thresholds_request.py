from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_thresholds_request_scope import LiveThresholdsRequestScope
from ..types import UNSET, Unset






T = TypeVar("T", bound="LiveThresholdsRequest")



@_attrs_define
class LiveThresholdsRequest:
    """ 
        Attributes:
            api_errors (float | Unset):
            bounce (float | Unset):
            complaint (float | Unset):
            deferral (float | Unset):
            mailbox_complaint (float | Unset):
            scope (LiveThresholdsRequestScope | Unset):
     """

    api_errors: float | Unset = UNSET
    bounce: float | Unset = UNSET
    complaint: float | Unset = UNSET
    deferral: float | Unset = UNSET
    mailbox_complaint: float | Unset = UNSET
    scope: LiveThresholdsRequestScope | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        api_errors = self.api_errors

        bounce = self.bounce

        complaint = self.complaint

        deferral = self.deferral

        mailbox_complaint = self.mailbox_complaint

        scope: str | Unset = UNSET
        if not isinstance(self.scope, Unset):
            scope = self.scope.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if api_errors is not UNSET:
            field_dict["apiErrors"] = api_errors
        if bounce is not UNSET:
            field_dict["bounce"] = bounce
        if complaint is not UNSET:
            field_dict["complaint"] = complaint
        if deferral is not UNSET:
            field_dict["deferral"] = deferral
        if mailbox_complaint is not UNSET:
            field_dict["mailboxComplaint"] = mailbox_complaint
        if scope is not UNSET:
            field_dict["scope"] = scope

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_errors = d.pop("apiErrors", UNSET)

        bounce = d.pop("bounce", UNSET)

        complaint = d.pop("complaint", UNSET)

        deferral = d.pop("deferral", UNSET)

        mailbox_complaint = d.pop("mailboxComplaint", UNSET)

        _scope = d.pop("scope", UNSET)
        scope: LiveThresholdsRequestScope | Unset
        if isinstance(_scope,  Unset):
            scope = UNSET
        else:
            scope = LiveThresholdsRequestScope(_scope)




        live_thresholds_request = cls(
            api_errors=api_errors,
            bounce=bounce,
            complaint=complaint,
            deferral=deferral,
            mailbox_complaint=mailbox_complaint,
            scope=scope,
        )

        return live_thresholds_request

