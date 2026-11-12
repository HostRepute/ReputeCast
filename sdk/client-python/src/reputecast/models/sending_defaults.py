from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sending_defaults_pause_scope import SendingDefaultsPauseScope
from ..models.sending_defaults_suppression_sync import SendingDefaultsSuppressionSync
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sending_defaults_throttles import SendingDefaultsThrottles





T = TypeVar("T", bound="SendingDefaults")



@_attrs_define
class SendingDefaults:
    """ 
        Attributes:
            bounce (float): Auto-pause at this hard-bounce rate, percent
            complaint (float): Auto-pause at this complaint rate, percent
            confirm_over (int): Sends to more recipients need the count typed to confirm
            mailbox_complaint (float): Auto-pause one mailbox provider at this complaint rate, percent
            pause_scope (SendingDefaultsPauseScope):
            provider_error (float): Treat a provider as failing at this API error rate over 3 minutes, percent
            suppression_sync (SendingDefaultsSuppressionSync):
            throttles (SendingDefaultsThrottles): Messages per second per mailbox provider (gmail, outlook, yahoo, icloud);
                0 = no limit
            secrets_rotated_at (datetime.datetime | Unset):
     """

    bounce: float
    complaint: float
    confirm_over: int
    mailbox_complaint: float
    pause_scope: SendingDefaultsPauseScope
    provider_error: float
    suppression_sync: SendingDefaultsSuppressionSync
    throttles: SendingDefaultsThrottles
    secrets_rotated_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sending_defaults_throttles import SendingDefaultsThrottles # noqa: PLC0415
        bounce = self.bounce

        complaint = self.complaint

        confirm_over = self.confirm_over

        mailbox_complaint = self.mailbox_complaint

        pause_scope = self.pause_scope.value

        provider_error = self.provider_error

        suppression_sync = self.suppression_sync.value

        throttles = self.throttles.to_dict()

        secrets_rotated_at: str | Unset = UNSET
        if not isinstance(self.secrets_rotated_at, Unset):
            secrets_rotated_at = self.secrets_rotated_at.isoformat()


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
        if secrets_rotated_at is not UNSET:
            field_dict["secretsRotatedAt"] = secrets_rotated_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sending_defaults_throttles import SendingDefaultsThrottles # noqa: PLC0415
        d = dict(src_dict)
        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        confirm_over = d.pop("confirmOver")

        mailbox_complaint = d.pop("mailboxComplaint")

        pause_scope = SendingDefaultsPauseScope(d.pop("pauseScope"))




        provider_error = d.pop("providerError")

        suppression_sync = SendingDefaultsSuppressionSync(d.pop("suppressionSync"))




        throttles = SendingDefaultsThrottles.from_dict(d.pop("throttles"))




        _secrets_rotated_at = d.pop("secretsRotatedAt", UNSET)
        secrets_rotated_at: datetime.datetime | Unset
        if isinstance(_secrets_rotated_at,  Unset):
            secrets_rotated_at = UNSET
        else:
            secrets_rotated_at = datetime.datetime.fromisoformat(_secrets_rotated_at)




        sending_defaults = cls(
            bounce=bounce,
            complaint=complaint,
            confirm_over=confirm_over,
            mailbox_complaint=mailbox_complaint,
            pause_scope=pause_scope,
            provider_error=provider_error,
            suppression_sync=suppression_sync,
            throttles=throttles,
            secrets_rotated_at=secrets_rotated_at,
        )

        return sending_defaults

