from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_rule_match_key import ApprovalRuleMatchKey
from ..types import UNSET, Unset






T = TypeVar("T", bound="ApprovalRuleMatch")



@_attrs_define
class ApprovalRuleMatch:
    """ 
        Attributes:
            badge (str):
            key (ApprovalRuleMatchKey):
            label (str):
            short (str):
            author (str | Unset):
            role (str | Unset):
     """

    badge: str
    key: ApprovalRuleMatchKey
    label: str
    short: str
    author: str | Unset = UNSET
    role: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        badge = self.badge

        key = self.key.value

        label = self.label

        short = self.short

        author = self.author

        role = self.role


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "badge": badge,
            "key": key,
            "label": label,
            "short": short,
        })
        if author is not UNSET:
            field_dict["author"] = author
        if role is not UNSET:
            field_dict["role"] = role

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        badge = d.pop("badge")

        key = ApprovalRuleMatchKey(d.pop("key"))




        label = d.pop("label")

        short = d.pop("short")

        author = d.pop("author", UNSET)

        role = d.pop("role", UNSET)

        approval_rule_match = cls(
            badge=badge,
            key=key,
            label=label,
            short=short,
            author=author,
            role=role,
        )

        return approval_rule_match

