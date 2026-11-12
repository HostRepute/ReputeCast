from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_relay_state import TransactionalRelayState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_relay_check import TransactionalRelayCheck





T = TypeVar("T", bound="TransactionalRelay")



@_attrs_define
class TransactionalRelay:
    """ 
        Attributes:
            admins (list[str]):
            can_manage (bool):
            checks (list[TransactionalRelayCheck]):
            enabled (bool):
            starttls_port (int):
            state (TransactionalRelayState):
            tls_port (int):
            cert_not_after (datetime.datetime | Unset):
            enabled_at (datetime.datetime | Unset):
            enabled_by (str | Unset):
            hostname (str | Unset):
            unreachable_since (datetime.datetime | Unset):
     """

    admins: list[str]
    can_manage: bool
    checks: list[TransactionalRelayCheck]
    enabled: bool
    starttls_port: int
    state: TransactionalRelayState
    tls_port: int
    cert_not_after: datetime.datetime | Unset = UNSET
    enabled_at: datetime.datetime | Unset = UNSET
    enabled_by: str | Unset = UNSET
    hostname: str | Unset = UNSET
    unreachable_since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_relay_check import TransactionalRelayCheck # noqa: PLC0415
        admins = self.admins



        can_manage = self.can_manage

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        enabled = self.enabled

        starttls_port = self.starttls_port

        state = self.state.value

        tls_port = self.tls_port

        cert_not_after: str | Unset = UNSET
        if not isinstance(self.cert_not_after, Unset):
            cert_not_after = self.cert_not_after.isoformat()

        enabled_at: str | Unset = UNSET
        if not isinstance(self.enabled_at, Unset):
            enabled_at = self.enabled_at.isoformat()

        enabled_by = self.enabled_by

        hostname = self.hostname

        unreachable_since: str | Unset = UNSET
        if not isinstance(self.unreachable_since, Unset):
            unreachable_since = self.unreachable_since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "admins": admins,
            "canManage": can_manage,
            "checks": checks,
            "enabled": enabled,
            "starttlsPort": starttls_port,
            "state": state,
            "tlsPort": tls_port,
        })
        if cert_not_after is not UNSET:
            field_dict["certNotAfter"] = cert_not_after
        if enabled_at is not UNSET:
            field_dict["enabledAt"] = enabled_at
        if enabled_by is not UNSET:
            field_dict["enabledBy"] = enabled_by
        if hostname is not UNSET:
            field_dict["hostname"] = hostname
        if unreachable_since is not UNSET:
            field_dict["unreachableSince"] = unreachable_since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_relay_check import TransactionalRelayCheck # noqa: PLC0415
        d = dict(src_dict)
        admins = cast(list[str], d.pop("admins"))


        can_manage = d.pop("canManage")

        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = TransactionalRelayCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        enabled = d.pop("enabled")

        starttls_port = d.pop("starttlsPort")

        state = TransactionalRelayState(d.pop("state"))




        tls_port = d.pop("tlsPort")

        _cert_not_after = d.pop("certNotAfter", UNSET)
        cert_not_after: datetime.datetime | Unset
        if isinstance(_cert_not_after,  Unset):
            cert_not_after = UNSET
        else:
            cert_not_after = datetime.datetime.fromisoformat(_cert_not_after)




        _enabled_at = d.pop("enabledAt", UNSET)
        enabled_at: datetime.datetime | Unset
        if isinstance(_enabled_at,  Unset):
            enabled_at = UNSET
        else:
            enabled_at = datetime.datetime.fromisoformat(_enabled_at)




        enabled_by = d.pop("enabledBy", UNSET)

        hostname = d.pop("hostname", UNSET)

        _unreachable_since = d.pop("unreachableSince", UNSET)
        unreachable_since: datetime.datetime | Unset
        if isinstance(_unreachable_since,  Unset):
            unreachable_since = UNSET
        else:
            unreachable_since = datetime.datetime.fromisoformat(_unreachable_since)




        transactional_relay = cls(
            admins=admins,
            can_manage=can_manage,
            checks=checks,
            enabled=enabled,
            starttls_port=starttls_port,
            state=state,
            tls_port=tls_port,
            cert_not_after=cert_not_after,
            enabled_at=enabled_at,
            enabled_by=enabled_by,
            hostname=hostname,
            unreachable_since=unreachable_since,
        )

        return transactional_relay

