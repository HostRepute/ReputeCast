from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.digest_rule_match import DigestRuleMatch





T = TypeVar("T", bound="DigestApprovals")



@_attrs_define
class DigestApprovals:
    """ 
        Attributes:
            approvers (list[str]):
            needed (bool):
            rules (list[DigestRuleMatch]):
     """

    approvers: list[str]
    needed: bool
    rules: list[DigestRuleMatch]





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_rule_match import DigestRuleMatch # noqa: PLC0415
        approvers = self.approvers



        needed = self.needed

        rules = []
        for rules_item_data in self.rules:
            rules_item = rules_item_data.to_dict()
            rules.append(rules_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "needed": needed,
            "rules": rules,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_rule_match import DigestRuleMatch # noqa: PLC0415
        d = dict(src_dict)
        approvers = cast(list[str], d.pop("approvers"))


        needed = d.pop("needed")

        rules = []
        _rules = d.pop("rules")
        for rules_item_data in (_rules):
            rules_item = DigestRuleMatch.from_dict(rules_item_data)



            rules.append(rules_item)


        digest_approvals = cls(
            approvers=approvers,
            needed=needed,
            rules=rules,
        )

        return digest_approvals

