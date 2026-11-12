from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.holdout_settings import HoldoutSettings
  from ..models.marketing_cap import MarketingCap
  from ..models.send_defaults import SendDefaults
  from ..models.sending_rule import SendingRule





T = TypeVar("T", bound="SendingRulesView")



@_attrs_define
class SendingRulesView:
    """ 
        Attributes:
            can_manage (bool):
            defaults (SendDefaults):
            holdout (HoldoutSettings):
            managers (list[str]):
            marketing_cap (MarketingCap):
            rules (list[SendingRule]):
            sunset_days (int):
            reason (str | Unset):
     """

    can_manage: bool
    defaults: SendDefaults
    holdout: HoldoutSettings
    managers: list[str]
    marketing_cap: MarketingCap
    rules: list[SendingRule]
    sunset_days: int
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.holdout_settings import HoldoutSettings # noqa: PLC0415
        from ..models.marketing_cap import MarketingCap # noqa: PLC0415
        from ..models.send_defaults import SendDefaults # noqa: PLC0415
        from ..models.sending_rule import SendingRule # noqa: PLC0415
        can_manage = self.can_manage

        defaults = self.defaults.to_dict()

        holdout = self.holdout.to_dict()

        managers = self.managers



        marketing_cap = self.marketing_cap.to_dict()

        rules = []
        for rules_item_data in self.rules:
            rules_item = rules_item_data.to_dict()
            rules.append(rules_item)



        sunset_days = self.sunset_days

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "defaults": defaults,
            "holdout": holdout,
            "managers": managers,
            "marketingCap": marketing_cap,
            "rules": rules,
            "sunsetDays": sunset_days,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.holdout_settings import HoldoutSettings # noqa: PLC0415
        from ..models.marketing_cap import MarketingCap # noqa: PLC0415
        from ..models.send_defaults import SendDefaults # noqa: PLC0415
        from ..models.sending_rule import SendingRule # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        defaults = SendDefaults.from_dict(d.pop("defaults"))




        holdout = HoldoutSettings.from_dict(d.pop("holdout"))




        managers = cast(list[str], d.pop("managers"))


        marketing_cap = MarketingCap.from_dict(d.pop("marketingCap"))




        rules = []
        _rules = d.pop("rules")
        for rules_item_data in (_rules):
            rules_item = SendingRule.from_dict(rules_item_data)



            rules.append(rules_item)


        sunset_days = d.pop("sunsetDays")

        reason = d.pop("reason", UNSET)

        sending_rules_view = cls(
            can_manage=can_manage,
            defaults=defaults,
            holdout=holdout,
            managers=managers,
            marketing_cap=marketing_cap,
            rules=rules,
            sunset_days=sunset_days,
            reason=reason,
        )

        return sending_rules_view

