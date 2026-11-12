from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.canary import Canary
  from ..models.traffic_reply_rule import TrafficReplyRule
  from ..models.traffic_thresholds import TrafficThresholds
  from ..models.watch import Watch





T = TypeVar("T", bound="TrafficRulesInput")



@_attrs_define
class TrafficRulesInput:
    """ 
        Attributes:
            canary (Canary):
            reply_rules (list[TrafficReplyRule]):
            thresholds (TrafficThresholds):
            watch (Watch):
     """

    canary: Canary
    reply_rules: list[TrafficReplyRule]
    thresholds: TrafficThresholds
    watch: Watch





    def to_dict(self) -> dict[str, Any]:
        from ..models.canary import Canary # noqa: PLC0415
        from ..models.traffic_reply_rule import TrafficReplyRule # noqa: PLC0415
        from ..models.traffic_thresholds import TrafficThresholds # noqa: PLC0415
        from ..models.watch import Watch # noqa: PLC0415
        canary = self.canary.to_dict()

        reply_rules = []
        for reply_rules_item_data in self.reply_rules:
            reply_rules_item = reply_rules_item_data.to_dict()
            reply_rules.append(reply_rules_item)



        thresholds = self.thresholds.to_dict()

        watch = self.watch.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canary": canary,
            "replyRules": reply_rules,
            "thresholds": thresholds,
            "watch": watch,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.canary import Canary # noqa: PLC0415
        from ..models.traffic_reply_rule import TrafficReplyRule # noqa: PLC0415
        from ..models.traffic_thresholds import TrafficThresholds # noqa: PLC0415
        from ..models.watch import Watch # noqa: PLC0415
        d = dict(src_dict)
        canary = Canary.from_dict(d.pop("canary"))




        reply_rules = []
        _reply_rules = d.pop("replyRules")
        for reply_rules_item_data in (_reply_rules):
            reply_rules_item = TrafficReplyRule.from_dict(reply_rules_item_data)



            reply_rules.append(reply_rules_item)


        thresholds = TrafficThresholds.from_dict(d.pop("thresholds"))




        watch = Watch.from_dict(d.pop("watch"))




        traffic_rules_input = cls(
            canary=canary,
            reply_rules=reply_rules,
            thresholds=thresholds,
            watch=watch,
        )

        return traffic_rules_input

