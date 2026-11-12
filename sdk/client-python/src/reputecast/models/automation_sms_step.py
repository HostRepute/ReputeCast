from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_sms_step_outside import AutomationSMSStepOutside
from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationSMSStep")



@_attrs_define
class AutomationSMSStep:
    """ 
        Attributes:
            body (str | Unset):
            consent_list_id (str | Unset):
            no_opt_out (bool | Unset):
            outside (AutomationSMSStepOutside | Unset):
            skip_recent_hours (int | Unset):
     """

    body: str | Unset = UNSET
    consent_list_id: str | Unset = UNSET
    no_opt_out: bool | Unset = UNSET
    outside: AutomationSMSStepOutside | Unset = UNSET
    skip_recent_hours: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        body = self.body

        consent_list_id = self.consent_list_id

        no_opt_out = self.no_opt_out

        outside: str | Unset = UNSET
        if not isinstance(self.outside, Unset):
            outside = self.outside.value


        skip_recent_hours = self.skip_recent_hours


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if body is not UNSET:
            field_dict["body"] = body
        if consent_list_id is not UNSET:
            field_dict["consentListId"] = consent_list_id
        if no_opt_out is not UNSET:
            field_dict["noOptOut"] = no_opt_out
        if outside is not UNSET:
            field_dict["outside"] = outside
        if skip_recent_hours is not UNSET:
            field_dict["skipRecentHours"] = skip_recent_hours

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        body = d.pop("body", UNSET)

        consent_list_id = d.pop("consentListId", UNSET)

        no_opt_out = d.pop("noOptOut", UNSET)

        _outside = d.pop("outside", UNSET)
        outside: AutomationSMSStepOutside | Unset
        if isinstance(_outside,  Unset):
            outside = UNSET
        else:
            outside = AutomationSMSStepOutside(_outside)




        skip_recent_hours = d.pop("skipRecentHours", UNSET)

        automation_sms_step = cls(
            body=body,
            consent_list_id=consent_list_id,
            no_opt_out=no_opt_out,
            outside=outside,
            skip_recent_hours=skip_recent_hours,
        )

        return automation_sms_step

