from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_step_stat import SignupFormStepStat





T = TypeVar("T", bound="SignupFormStats")



@_attrs_define
class SignupFormStats:
    """ 
        Attributes:
            blocked (int): Submissions blocked as spam
            confirmed (int):
            confirmed_rate (float):
            days (int):
            pending (int): Double opt-in confirmations not clicked yet
            step_signups (int):
            steps (list[SignupFormStepStat]):
            submission_rate (float):
            submissions (int):
            views (int):
     """

    blocked: int
    confirmed: int
    confirmed_rate: float
    days: int
    pending: int
    step_signups: int
    steps: list[SignupFormStepStat]
    submission_rate: float
    submissions: int
    views: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_step_stat import SignupFormStepStat # noqa: PLC0415
        blocked = self.blocked

        confirmed = self.confirmed

        confirmed_rate = self.confirmed_rate

        days = self.days

        pending = self.pending

        step_signups = self.step_signups

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)



        submission_rate = self.submission_rate

        submissions = self.submissions

        views = self.views


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocked": blocked,
            "confirmed": confirmed,
            "confirmedRate": confirmed_rate,
            "days": days,
            "pending": pending,
            "stepSignups": step_signups,
            "steps": steps,
            "submissionRate": submission_rate,
            "submissions": submissions,
            "views": views,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_step_stat import SignupFormStepStat # noqa: PLC0415
        d = dict(src_dict)
        blocked = d.pop("blocked")

        confirmed = d.pop("confirmed")

        confirmed_rate = d.pop("confirmedRate")

        days = d.pop("days")

        pending = d.pop("pending")

        step_signups = d.pop("stepSignups")

        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = SignupFormStepStat.from_dict(steps_item_data)



            steps.append(steps_item)


        submission_rate = d.pop("submissionRate")

        submissions = d.pop("submissions")

        views = d.pop("views")

        signup_form_stats = cls(
            blocked=blocked,
            confirmed=confirmed,
            confirmed_rate=confirmed_rate,
            days=days,
            pending=pending,
            step_signups=step_signups,
            steps=steps,
            submission_rate=submission_rate,
            submissions=submissions,
            views=views,
        )

        return signup_form_stats

