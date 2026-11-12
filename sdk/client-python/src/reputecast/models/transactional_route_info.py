from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalRouteInfo")



@_attrs_define
class TransactionalRouteInfo:
    """ 
        Attributes:
            failed_over (bool):
            rule (str):
            fallback (str | Unset):
            matched (str | Unset): The condition's field: category or mailbox_provider
            policy (str | Unset):
            primary (str | Unset):
            value (str | Unset):
     """

    failed_over: bool
    rule: str
    fallback: str | Unset = UNSET
    matched: str | Unset = UNSET
    policy: str | Unset = UNSET
    primary: str | Unset = UNSET
    value: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        failed_over = self.failed_over

        rule = self.rule

        fallback = self.fallback

        matched = self.matched

        policy = self.policy

        primary = self.primary

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "failedOver": failed_over,
            "rule": rule,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if matched is not UNSET:
            field_dict["matched"] = matched
        if policy is not UNSET:
            field_dict["policy"] = policy
        if primary is not UNSET:
            field_dict["primary"] = primary
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        failed_over = d.pop("failedOver")

        rule = d.pop("rule")

        fallback = d.pop("fallback", UNSET)

        matched = d.pop("matched", UNSET)

        policy = d.pop("policy", UNSET)

        primary = d.pop("primary", UNSET)

        value = d.pop("value", UNSET)

        transactional_route_info = cls(
            failed_over=failed_over,
            rule=rule,
            fallback=fallback,
            matched=matched,
            policy=policy,
            primary=primary,
            value=value,
        )

        return transactional_route_info

