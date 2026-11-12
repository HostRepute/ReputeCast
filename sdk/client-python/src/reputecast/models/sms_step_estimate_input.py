from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SmsStepEstimateInput")



@_attrs_define
class SmsStepEstimateInput:
    """ 
        Attributes:
            body (str):
            consent_list_id (str | Unset):
            no_opt_out (bool | Unset):
     """

    body: str
    consent_list_id: str | Unset = UNSET
    no_opt_out: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        body = self.body

        consent_list_id = self.consent_list_id

        no_opt_out = self.no_opt_out


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
        })
        if consent_list_id is not UNSET:
            field_dict["consentListId"] = consent_list_id
        if no_opt_out is not UNSET:
            field_dict["noOptOut"] = no_opt_out

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        body = d.pop("body")

        consent_list_id = d.pop("consentListId", UNSET)

        no_opt_out = d.pop("noOptOut", UNSET)

        sms_step_estimate_input = cls(
            body=body,
            consent_list_id=consent_list_id,
            no_opt_out=no_opt_out,
        )

        return sms_step_estimate_input

