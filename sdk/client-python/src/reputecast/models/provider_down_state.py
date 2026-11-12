from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_down_state_reason import ProviderDownStateReason
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.provider_call_error import ProviderCallError





T = TypeVar("T", bound="ProviderDownState")



@_attrs_define
class ProviderDownState:
    """ 
        Attributes:
            errors (list[ProviderCallError]):
            reason (ProviderDownStateReason):
            since (datetime.datetime):
            credential_label (str | Unset):
            last_success_at (datetime.datetime | Unset):
            last_success_op (str | Unset):
            status_label (str | Unset):
            status_summary (str | Unset):
            status_url (str | Unset):
     """

    errors: list[ProviderCallError]
    reason: ProviderDownStateReason
    since: datetime.datetime
    credential_label: str | Unset = UNSET
    last_success_at: datetime.datetime | Unset = UNSET
    last_success_op: str | Unset = UNSET
    status_label: str | Unset = UNSET
    status_summary: str | Unset = UNSET
    status_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_call_error import ProviderCallError # noqa: PLC0415
        errors = []
        for errors_item_data in self.errors:
            errors_item = errors_item_data.to_dict()
            errors.append(errors_item)



        reason = self.reason.value

        since = self.since.isoformat()

        credential_label = self.credential_label

        last_success_at: str | Unset = UNSET
        if not isinstance(self.last_success_at, Unset):
            last_success_at = self.last_success_at.isoformat()

        last_success_op = self.last_success_op

        status_label = self.status_label

        status_summary = self.status_summary

        status_url = self.status_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "errors": errors,
            "reason": reason,
            "since": since,
        })
        if credential_label is not UNSET:
            field_dict["credentialLabel"] = credential_label
        if last_success_at is not UNSET:
            field_dict["lastSuccessAt"] = last_success_at
        if last_success_op is not UNSET:
            field_dict["lastSuccessOp"] = last_success_op
        if status_label is not UNSET:
            field_dict["statusLabel"] = status_label
        if status_summary is not UNSET:
            field_dict["statusSummary"] = status_summary
        if status_url is not UNSET:
            field_dict["statusUrl"] = status_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_call_error import ProviderCallError # noqa: PLC0415
        d = dict(src_dict)
        errors = []
        _errors = d.pop("errors")
        for errors_item_data in (_errors):
            errors_item = ProviderCallError.from_dict(errors_item_data)



            errors.append(errors_item)


        reason = ProviderDownStateReason(d.pop("reason"))




        since = datetime.datetime.fromisoformat(d.pop("since"))




        credential_label = d.pop("credentialLabel", UNSET)

        _last_success_at = d.pop("lastSuccessAt", UNSET)
        last_success_at: datetime.datetime | Unset
        if isinstance(_last_success_at,  Unset):
            last_success_at = UNSET
        else:
            last_success_at = datetime.datetime.fromisoformat(_last_success_at)




        last_success_op = d.pop("lastSuccessOp", UNSET)

        status_label = d.pop("statusLabel", UNSET)

        status_summary = d.pop("statusSummary", UNSET)

        status_url = d.pop("statusUrl", UNSET)

        provider_down_state = cls(
            errors=errors,
            reason=reason,
            since=since,
            credential_label=credential_label,
            last_success_at=last_success_at,
            last_success_op=last_success_op,
            status_label=status_label,
            status_summary=status_summary,
            status_url=status_url,
        )

        return provider_down_state

