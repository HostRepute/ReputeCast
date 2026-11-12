from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_item_state_kind import IntegrationItemStateKind
from ..models.integration_item_state_off_reason import IntegrationItemStateOffReason
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="IntegrationItemState")



@_attrs_define
class IntegrationItemState:
    """ 
        Attributes:
            available (bool):
            key (str):
            kind (IntegrationItemStateKind):
            new (bool):
            on (bool):
            permissions (list[str]):
            sensitive (bool):
            event (str | Unset):
            missing (str | Unset):
            off_reason (IntegrationItemStateOffReason | Unset):
            owner (str | Unset):
     """

    available: bool
    key: str
    kind: IntegrationItemStateKind
    new: bool
    on: bool
    permissions: list[str]
    sensitive: bool
    event: str | Unset = UNSET
    missing: str | Unset = UNSET
    off_reason: IntegrationItemStateOffReason | Unset = UNSET
    owner: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        key = self.key

        kind = self.kind.value

        new = self.new

        on = self.on

        permissions = self.permissions



        sensitive = self.sensitive

        event = self.event

        missing = self.missing

        off_reason: str | Unset = UNSET
        if not isinstance(self.off_reason, Unset):
            off_reason = self.off_reason.value


        owner = self.owner


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "key": key,
            "kind": kind,
            "new": new,
            "on": on,
            "permissions": permissions,
            "sensitive": sensitive,
        })
        if event is not UNSET:
            field_dict["event"] = event
        if missing is not UNSET:
            field_dict["missing"] = missing
        if off_reason is not UNSET:
            field_dict["offReason"] = off_reason
        if owner is not UNSET:
            field_dict["owner"] = owner

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        key = d.pop("key")

        kind = IntegrationItemStateKind(d.pop("kind"))




        new = d.pop("new")

        on = d.pop("on")

        permissions = cast(list[str], d.pop("permissions"))


        sensitive = d.pop("sensitive")

        event = d.pop("event", UNSET)

        missing = d.pop("missing", UNSET)

        _off_reason = d.pop("offReason", UNSET)
        off_reason: IntegrationItemStateOffReason | Unset
        if isinstance(_off_reason,  Unset):
            off_reason = UNSET
        else:
            off_reason = IntegrationItemStateOffReason(_off_reason)




        owner = d.pop("owner", UNSET)

        integration_item_state = cls(
            available=available,
            key=key,
            kind=kind,
            new=new,
            on=on,
            permissions=permissions,
            sensitive=sensitive,
            event=event,
            missing=missing,
            off_reason=off_reason,
            owner=owner,
        )

        return integration_item_state

