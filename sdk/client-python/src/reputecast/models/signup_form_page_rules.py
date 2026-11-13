from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_page_rules_mode import SignupFormPageRulesMode
from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_page_rule import SignupFormPageRule





T = TypeVar("T", bound="SignupFormPageRules")



@_attrs_define
class SignupFormPageRules:
    """ 
        Attributes:
            mode (SignupFormPageRulesMode):
            rules (list[SignupFormPageRule]):
     """

    mode: SignupFormPageRulesMode
    rules: list[SignupFormPageRule]





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_page_rule import SignupFormPageRule # noqa: PLC0415
        mode = self.mode.value

        rules = []
        for rules_item_data in self.rules:
            rules_item = rules_item_data.to_dict()
            rules.append(rules_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "rules": rules,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_page_rule import SignupFormPageRule # noqa: PLC0415
        d = dict(src_dict)
        mode = SignupFormPageRulesMode(d.pop("mode"))




        rules = []
        _rules = d.pop("rules")
        for rules_item_data in (_rules):
            rules_item = SignupFormPageRule.from_dict(rules_item_data)



            rules.append(rules_item)


        signup_form_page_rules = cls(
            mode=mode,
            rules=rules,
        )

        return signup_form_page_rules

