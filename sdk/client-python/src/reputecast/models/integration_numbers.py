from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationNumbers")



@_attrs_define
class IntegrationNumbers:
    """ 
        Attributes:
            calls7d (int):
            consent_kept_7_d (int): Calls refused to keep consent (409)
            errors7d (int):
            fixed7d (int): Errors a later call from the same workflow got past
            managed_webhooks (int):
            last_call_at (datetime.datetime | Unset):
            last_call_workflow (str | Unset):
     """

    calls7d: int
    consent_kept_7_d: int
    errors7d: int
    fixed7d: int
    managed_webhooks: int
    last_call_at: datetime.datetime | Unset = UNSET
    last_call_workflow: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        calls7d = self.calls7d

        consent_kept_7_d = self.consent_kept_7_d

        errors7d = self.errors7d

        fixed7d = self.fixed7d

        managed_webhooks = self.managed_webhooks

        last_call_at: str | Unset = UNSET
        if not isinstance(self.last_call_at, Unset):
            last_call_at = self.last_call_at.isoformat()

        last_call_workflow = self.last_call_workflow


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "calls7d": calls7d,
            "consentKept7d": consent_kept_7_d,
            "errors7d": errors7d,
            "fixed7d": fixed7d,
            "managedWebhooks": managed_webhooks,
        })
        if last_call_at is not UNSET:
            field_dict["lastCallAt"] = last_call_at
        if last_call_workflow is not UNSET:
            field_dict["lastCallWorkflow"] = last_call_workflow

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        calls7d = d.pop("calls7d")

        consent_kept_7_d = d.pop("consentKept7d")

        errors7d = d.pop("errors7d")

        fixed7d = d.pop("fixed7d")

        managed_webhooks = d.pop("managedWebhooks")

        _last_call_at = d.pop("lastCallAt", UNSET)
        last_call_at: datetime.datetime | Unset
        if isinstance(_last_call_at,  Unset):
            last_call_at = UNSET
        else:
            last_call_at = datetime.datetime.fromisoformat(_last_call_at)




        last_call_workflow = d.pop("lastCallWorkflow", UNSET)

        integration_numbers = cls(
            calls7d=calls7d,
            consent_kept_7_d=consent_kept_7_d,
            errors7d=errors7d,
            fixed7d=fixed7d,
            managed_webhooks=managed_webhooks,
            last_call_at=last_call_at,
            last_call_workflow=last_call_workflow,
        )

        return integration_numbers

