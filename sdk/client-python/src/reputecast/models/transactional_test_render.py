from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_test_render_source import TransactionalTestRenderSource
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="TransactionalTestRender")



@_attrs_define
class TransactionalTestRender:
    """ 
        Attributes:
            defaults (list[str]):
            missing (list[str]):
            name (str):
            ok (bool):
            source (TransactionalTestRenderSource):
            at (str | Unset):
            problem (str | Unset):
            recipient (str | Unset):
            subject (str | Unset):
     """

    defaults: list[str]
    missing: list[str]
    name: str
    ok: bool
    source: TransactionalTestRenderSource
    at: str | Unset = UNSET
    problem: str | Unset = UNSET
    recipient: str | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        defaults = self.defaults



        missing = self.missing



        name = self.name

        ok = self.ok

        source = self.source.value

        at = self.at

        problem = self.problem

        recipient = self.recipient

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "defaults": defaults,
            "missing": missing,
            "name": name,
            "ok": ok,
            "source": source,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if problem is not UNSET:
            field_dict["problem"] = problem
        if recipient is not UNSET:
            field_dict["recipient"] = recipient
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        defaults = cast(list[str], d.pop("defaults"))


        missing = cast(list[str], d.pop("missing"))


        name = d.pop("name")

        ok = d.pop("ok")

        source = TransactionalTestRenderSource(d.pop("source"))




        at = d.pop("at", UNSET)

        problem = d.pop("problem", UNSET)

        recipient = d.pop("recipient", UNSET)

        subject = d.pop("subject", UNSET)

        transactional_test_render = cls(
            defaults=defaults,
            missing=missing,
            name=name,
            ok=ok,
            source=source,
            at=at,
            problem=problem,
            recipient=recipient,
            subject=subject,
        )

        return transactional_test_render

