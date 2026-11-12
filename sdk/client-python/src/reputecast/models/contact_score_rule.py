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
  from ..models.scoring_rule import ScoringRule





T = TypeVar("T", bound="ContactScoreRule")



@_attrs_define
class ContactScoreRule:
    """ 
        Attributes:
            at_max (bool):
            count (int):
            number (int): The rule's number on Scoring
            points (int):
            rule (ScoringRule):
            cap_reached (datetime.datetime | Unset):
            form_name (str | Unset):
            latest (datetime.datetime | Unset):
            list_name (str | Unset):
            zero_at (datetime.datetime | Unset):
            zero_email (str | Unset):
     """

    at_max: bool
    count: int
    number: int
    points: int
    rule: ScoringRule
    cap_reached: datetime.datetime | Unset = UNSET
    form_name: str | Unset = UNSET
    latest: datetime.datetime | Unset = UNSET
    list_name: str | Unset = UNSET
    zero_at: datetime.datetime | Unset = UNSET
    zero_email: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_rule import ScoringRule # noqa: PLC0415
        at_max = self.at_max

        count = self.count

        number = self.number

        points = self.points

        rule = self.rule.to_dict()

        cap_reached: str | Unset = UNSET
        if not isinstance(self.cap_reached, Unset):
            cap_reached = self.cap_reached.isoformat()

        form_name = self.form_name

        latest: str | Unset = UNSET
        if not isinstance(self.latest, Unset):
            latest = self.latest.isoformat()

        list_name = self.list_name

        zero_at: str | Unset = UNSET
        if not isinstance(self.zero_at, Unset):
            zero_at = self.zero_at.isoformat()

        zero_email = self.zero_email


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "atMax": at_max,
            "count": count,
            "number": number,
            "points": points,
            "rule": rule,
        })
        if cap_reached is not UNSET:
            field_dict["capReached"] = cap_reached
        if form_name is not UNSET:
            field_dict["formName"] = form_name
        if latest is not UNSET:
            field_dict["latest"] = latest
        if list_name is not UNSET:
            field_dict["listName"] = list_name
        if zero_at is not UNSET:
            field_dict["zeroAt"] = zero_at
        if zero_email is not UNSET:
            field_dict["zeroEmail"] = zero_email

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_rule import ScoringRule # noqa: PLC0415
        d = dict(src_dict)
        at_max = d.pop("atMax")

        count = d.pop("count")

        number = d.pop("number")

        points = d.pop("points")

        rule = ScoringRule.from_dict(d.pop("rule"))




        _cap_reached = d.pop("capReached", UNSET)
        cap_reached: datetime.datetime | Unset
        if isinstance(_cap_reached,  Unset):
            cap_reached = UNSET
        else:
            cap_reached = datetime.datetime.fromisoformat(_cap_reached)




        form_name = d.pop("formName", UNSET)

        _latest = d.pop("latest", UNSET)
        latest: datetime.datetime | Unset
        if isinstance(_latest,  Unset):
            latest = UNSET
        else:
            latest = datetime.datetime.fromisoformat(_latest)




        list_name = d.pop("listName", UNSET)

        _zero_at = d.pop("zeroAt", UNSET)
        zero_at: datetime.datetime | Unset
        if isinstance(_zero_at,  Unset):
            zero_at = UNSET
        else:
            zero_at = datetime.datetime.fromisoformat(_zero_at)




        zero_email = d.pop("zeroEmail", UNSET)

        contact_score_rule = cls(
            at_max=at_max,
            count=count,
            number=number,
            points=points,
            rule=rule,
            cap_reached=cap_reached,
            form_name=form_name,
            latest=latest,
            list_name=list_name,
            zero_at=zero_at,
            zero_email=zero_email,
        )

        return contact_score_rule

