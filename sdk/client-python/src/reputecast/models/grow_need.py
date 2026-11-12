from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.grow_need_kind import GrowNeedKind
from ..models.grow_need_reason import GrowNeedReason
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.grow_named_count import GrowNamedCount





T = TypeVar("T", bound="GrowNeed")



@_attrs_define
class GrowNeed:
    """ 
        Attributes:
            at (datetime.datetime):
            codes_left (int):
            count (int):
            expires (list[datetime.datetime]):
            kind (GrowNeedKind):
            milestone (int):
            milestone_id (str):
            partner (str):
            people (list[GrowNamedCount]):
            reason (GrowNeedReason):
            reward (str):
            since (datetime.datetime | Unset):
     """

    at: datetime.datetime
    codes_left: int
    count: int
    expires: list[datetime.datetime]
    kind: GrowNeedKind
    milestone: int
    milestone_id: str
    partner: str
    people: list[GrowNamedCount]
    reason: GrowNeedReason
    reward: str
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_named_count import GrowNamedCount # noqa: PLC0415
        at = self.at.isoformat()

        codes_left = self.codes_left

        count = self.count

        expires = []
        for expires_item_data in self.expires:
            expires_item = expires_item_data.isoformat()
            expires.append(expires_item)



        kind = self.kind.value

        milestone = self.milestone

        milestone_id = self.milestone_id

        partner = self.partner

        people = []
        for people_item_data in self.people:
            people_item = people_item_data.to_dict()
            people.append(people_item)



        reason = self.reason.value

        reward = self.reward

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "codesLeft": codes_left,
            "count": count,
            "expires": expires,
            "kind": kind,
            "milestone": milestone,
            "milestoneId": milestone_id,
            "partner": partner,
            "people": people,
            "reason": reason,
            "reward": reward,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_named_count import GrowNamedCount # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        codes_left = d.pop("codesLeft")

        count = d.pop("count")

        expires = []
        _expires = d.pop("expires")
        for expires_item_data in (_expires):
            expires_item = datetime.datetime.fromisoformat(expires_item_data)



            expires.append(expires_item)


        kind = GrowNeedKind(d.pop("kind"))




        milestone = d.pop("milestone")

        milestone_id = d.pop("milestoneId")

        partner = d.pop("partner")

        people = []
        _people = d.pop("people")
        for people_item_data in (_people):
            people_item = GrowNamedCount.from_dict(people_item_data)



            people.append(people_item)


        reason = GrowNeedReason(d.pop("reason"))




        reward = d.pop("reward")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        grow_need = cls(
            at=at,
            codes_left=codes_left,
            count=count,
            expires=expires,
            kind=kind,
            milestone=milestone,
            milestone_id=milestone_id,
            partner=partner,
            people=people,
            reason=reason,
            reward=reward,
            since=since,
        )

        return grow_need

