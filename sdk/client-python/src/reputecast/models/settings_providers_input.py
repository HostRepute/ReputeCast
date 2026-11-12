from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.settings_providers_input_pause_scope import SettingsProvidersInputPauseScope
from ..models.settings_providers_input_suppression_sync import SettingsProvidersInputSuppressionSync
from typing import cast

if TYPE_CHECKING:
  from ..models.settings_providers_input_throttles import SettingsProvidersInputThrottles





T = TypeVar("T", bound="SettingsProvidersInput")



@_attrs_define
class SettingsProvidersInput:
    """ 
        Attributes:
            bounce (float):
            complaint (float):
            confirm_over (int):
            mailbox_complaint (float):
            pause_scope (SettingsProvidersInputPauseScope):
            provider_error (float):
            suppression_sync (SettingsProvidersInputSuppressionSync):
            throttles (SettingsProvidersInputThrottles):
     """

    bounce: float
    complaint: float
    confirm_over: int
    mailbox_complaint: float
    pause_scope: SettingsProvidersInputPauseScope
    provider_error: float
    suppression_sync: SettingsProvidersInputSuppressionSync
    throttles: SettingsProvidersInputThrottles





    def to_dict(self) -> dict[str, Any]:
        from ..models.settings_providers_input_throttles import SettingsProvidersInputThrottles # noqa: PLC0415
        bounce = self.bounce

        complaint = self.complaint

        confirm_over = self.confirm_over

        mailbox_complaint = self.mailbox_complaint

        pause_scope = self.pause_scope.value

        provider_error = self.provider_error

        suppression_sync = self.suppression_sync.value

        throttles = self.throttles.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounce": bounce,
            "complaint": complaint,
            "confirmOver": confirm_over,
            "mailboxComplaint": mailbox_complaint,
            "pauseScope": pause_scope,
            "providerError": provider_error,
            "suppressionSync": suppression_sync,
            "throttles": throttles,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.settings_providers_input_throttles import SettingsProvidersInputThrottles # noqa: PLC0415
        d = dict(src_dict)
        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        confirm_over = d.pop("confirmOver")

        mailbox_complaint = d.pop("mailboxComplaint")

        pause_scope = SettingsProvidersInputPauseScope(d.pop("pauseScope"))




        provider_error = d.pop("providerError")

        suppression_sync = SettingsProvidersInputSuppressionSync(d.pop("suppressionSync"))




        throttles = SettingsProvidersInputThrottles.from_dict(d.pop("throttles"))




        settings_providers_input = cls(
            bounce=bounce,
            complaint=complaint,
            confirm_over=confirm_over,
            mailbox_complaint=mailbox_complaint,
            pause_scope=pause_scope,
            provider_error=provider_error,
            suppression_sync=suppression_sync,
            throttles=throttles,
        )

        return settings_providers_input

