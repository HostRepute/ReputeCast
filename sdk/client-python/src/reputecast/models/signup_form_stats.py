from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







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
            submission_rate (float):
            submissions (int):
            views (int):
     """

    blocked: int
    confirmed: int
    confirmed_rate: float
    days: int
    pending: int
    submission_rate: float
    submissions: int
    views: int





    def to_dict(self) -> dict[str, Any]:
        blocked = self.blocked

        confirmed = self.confirmed

        confirmed_rate = self.confirmed_rate

        days = self.days

        pending = self.pending

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
            "submissionRate": submission_rate,
            "submissions": submissions,
            "views": views,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        blocked = d.pop("blocked")

        confirmed = d.pop("confirmed")

        confirmed_rate = d.pop("confirmedRate")

        days = d.pop("days")

        pending = d.pop("pending")

        submission_rate = d.pop("submissionRate")

        submissions = d.pop("submissions")

        views = d.pop("views")

        signup_form_stats = cls(
            blocked=blocked,
            confirmed=confirmed,
            confirmed_rate=confirmed_rate,
            days=days,
            pending=pending,
            submission_rate=submission_rate,
            submissions=submissions,
            views=views,
        )

        return signup_form_stats

