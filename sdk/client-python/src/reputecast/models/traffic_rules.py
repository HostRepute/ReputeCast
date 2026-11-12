from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.canary import Canary
  from ..models.traffic_action import TrafficAction
  from ..models.traffic_reply_rule import TrafficReplyRule
  from ..models.traffic_thresholds import TrafficThresholds
  from ..models.watch import Watch





T = TypeVar("T", bound="TrafficRules")



@_attrs_define
class TrafficRules:
    """ 
        Attributes:
            can_manage (bool):
            canary (Canary):
            recent (list[TrafficAction]):
            reply_rules (list[TrafficReplyRule]):
            resumers (list[str]):
            seeds (int):
            thresholds (TrafficThresholds):
            usual (int):
            watch (Watch):
            saved_at (datetime.datetime | Unset):
            saved_by (str | Unset):
     """

    can_manage: bool
    canary: Canary
    recent: list[TrafficAction]
    reply_rules: list[TrafficReplyRule]
    resumers: list[str]
    seeds: int
    thresholds: TrafficThresholds
    usual: int
    watch: Watch
    saved_at: datetime.datetime | Unset = UNSET
    saved_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.canary import Canary # noqa: PLC0415
        from ..models.traffic_action import TrafficAction # noqa: PLC0415
        from ..models.traffic_reply_rule import TrafficReplyRule # noqa: PLC0415
        from ..models.traffic_thresholds import TrafficThresholds # noqa: PLC0415
        from ..models.watch import Watch # noqa: PLC0415
        can_manage = self.can_manage

        canary = self.canary.to_dict()

        recent = []
        for recent_item_data in self.recent:
            recent_item = recent_item_data.to_dict()
            recent.append(recent_item)



        reply_rules = []
        for reply_rules_item_data in self.reply_rules:
            reply_rules_item = reply_rules_item_data.to_dict()
            reply_rules.append(reply_rules_item)



        resumers = self.resumers



        seeds = self.seeds

        thresholds = self.thresholds.to_dict()

        usual = self.usual

        watch = self.watch.to_dict()

        saved_at: str | Unset = UNSET
        if not isinstance(self.saved_at, Unset):
            saved_at = self.saved_at.isoformat()

        saved_by = self.saved_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "canary": canary,
            "recent": recent,
            "replyRules": reply_rules,
            "resumers": resumers,
            "seeds": seeds,
            "thresholds": thresholds,
            "usual": usual,
            "watch": watch,
        })
        if saved_at is not UNSET:
            field_dict["savedAt"] = saved_at
        if saved_by is not UNSET:
            field_dict["savedBy"] = saved_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.canary import Canary # noqa: PLC0415
        from ..models.traffic_action import TrafficAction # noqa: PLC0415
        from ..models.traffic_reply_rule import TrafficReplyRule # noqa: PLC0415
        from ..models.traffic_thresholds import TrafficThresholds # noqa: PLC0415
        from ..models.watch import Watch # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        canary = Canary.from_dict(d.pop("canary"))




        recent = []
        _recent = d.pop("recent")
        for recent_item_data in (_recent):
            recent_item = TrafficAction.from_dict(recent_item_data)



            recent.append(recent_item)


        reply_rules = []
        _reply_rules = d.pop("replyRules")
        for reply_rules_item_data in (_reply_rules):
            reply_rules_item = TrafficReplyRule.from_dict(reply_rules_item_data)



            reply_rules.append(reply_rules_item)


        resumers = cast(list[str], d.pop("resumers"))


        seeds = d.pop("seeds")

        thresholds = TrafficThresholds.from_dict(d.pop("thresholds"))




        usual = d.pop("usual")

        watch = Watch.from_dict(d.pop("watch"))




        _saved_at = d.pop("savedAt", UNSET)
        saved_at: datetime.datetime | Unset
        if isinstance(_saved_at,  Unset):
            saved_at = UNSET
        else:
            saved_at = datetime.datetime.fromisoformat(_saved_at)




        saved_by = d.pop("savedBy", UNSET)

        traffic_rules = cls(
            can_manage=can_manage,
            canary=canary,
            recent=recent,
            reply_rules=reply_rules,
            resumers=resumers,
            seeds=seeds,
            thresholds=thresholds,
            usual=usual,
            watch=watch,
            saved_at=saved_at,
            saved_by=saved_by,
        )

        return traffic_rules

