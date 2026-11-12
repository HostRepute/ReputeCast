from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_route_summary_policy import TransactionalRouteSummaryPolicy
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_ref import TransactionalRef





T = TypeVar("T", bound="TransactionalRouteSummary")



@_attrs_define
class TransactionalRouteSummary:
    """ 
        Attributes:
            conditions (int):
            configured (bool):
            priority (bool):
            fallback (TransactionalRef | Unset):
            policy (TransactionalRouteSummaryPolicy | Unset):
            primary (TransactionalRef | Unset):
            system (TransactionalRef | Unset):
     """

    conditions: int
    configured: bool
    priority: bool
    fallback: TransactionalRef | Unset = UNSET
    policy: TransactionalRouteSummaryPolicy | Unset = UNSET
    primary: TransactionalRef | Unset = UNSET
    system: TransactionalRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        conditions = self.conditions

        configured = self.configured

        priority = self.priority

        fallback: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fallback, Unset):
            fallback = self.fallback.to_dict()

        policy: str | Unset = UNSET
        if not isinstance(self.policy, Unset):
            policy = self.policy.value


        primary: dict[str, Any] | Unset = UNSET
        if not isinstance(self.primary, Unset):
            primary = self.primary.to_dict()

        system: dict[str, Any] | Unset = UNSET
        if not isinstance(self.system, Unset):
            system = self.system.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conditions": conditions,
            "configured": configured,
            "priority": priority,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if policy is not UNSET:
            field_dict["policy"] = policy
        if primary is not UNSET:
            field_dict["primary"] = primary
        if system is not UNSET:
            field_dict["system"] = system

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        d = dict(src_dict)
        conditions = d.pop("conditions")

        configured = d.pop("configured")

        priority = d.pop("priority")

        _fallback = d.pop("fallback", UNSET)
        fallback: TransactionalRef | Unset
        if isinstance(_fallback,  Unset):
            fallback = UNSET
        else:
            fallback = TransactionalRef.from_dict(_fallback)




        _policy = d.pop("policy", UNSET)
        policy: TransactionalRouteSummaryPolicy | Unset
        if isinstance(_policy,  Unset):
            policy = UNSET
        else:
            policy = TransactionalRouteSummaryPolicy(_policy)




        _primary = d.pop("primary", UNSET)
        primary: TransactionalRef | Unset
        if isinstance(_primary,  Unset):
            primary = UNSET
        else:
            primary = TransactionalRef.from_dict(_primary)




        _system = d.pop("system", UNSET)
        system: TransactionalRef | Unset
        if isinstance(_system,  Unset):
            system = UNSET
        else:
            system = TransactionalRef.from_dict(_system)




        transactional_route_summary = cls(
            conditions=conditions,
            configured=configured,
            priority=priority,
            fallback=fallback,
            policy=policy,
            primary=primary,
            system=system,
        )

        return transactional_route_summary

