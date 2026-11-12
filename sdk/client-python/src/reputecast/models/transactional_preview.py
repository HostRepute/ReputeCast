from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="TransactionalPreview")



@_attrs_define
class TransactionalPreview:
    """ 
        Attributes:
            defaults (list[str]):
            from_email (str):
            from_name (str):
            html (str):
            missing (list[str]):
            ok (bool):
            preheader (str):
            reply_to (str):
            size_bytes (int):
            subject (str):
            text (str):
            version (str):
            problem (str | Unset):
     """

    defaults: list[str]
    from_email: str
    from_name: str
    html: str
    missing: list[str]
    ok: bool
    preheader: str
    reply_to: str
    size_bytes: int
    subject: str
    text: str
    version: str
    problem: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        defaults = self.defaults



        from_email = self.from_email

        from_name = self.from_name

        html = self.html

        missing = self.missing



        ok = self.ok

        preheader = self.preheader

        reply_to = self.reply_to

        size_bytes = self.size_bytes

        subject = self.subject

        text = self.text

        version = self.version

        problem = self.problem


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "defaults": defaults,
            "fromEmail": from_email,
            "fromName": from_name,
            "html": html,
            "missing": missing,
            "ok": ok,
            "preheader": preheader,
            "replyTo": reply_to,
            "sizeBytes": size_bytes,
            "subject": subject,
            "text": text,
            "version": version,
        })
        if problem is not UNSET:
            field_dict["problem"] = problem

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        defaults = cast(list[str], d.pop("defaults"))


        from_email = d.pop("fromEmail")

        from_name = d.pop("fromName")

        html = d.pop("html")

        missing = cast(list[str], d.pop("missing"))


        ok = d.pop("ok")

        preheader = d.pop("preheader")

        reply_to = d.pop("replyTo")

        size_bytes = d.pop("sizeBytes")

        subject = d.pop("subject")

        text = d.pop("text")

        version = d.pop("version")

        problem = d.pop("problem", UNSET)

        transactional_preview = cls(
            defaults=defaults,
            from_email=from_email,
            from_name=from_name,
            html=html,
            missing=missing,
            ok=ok,
            preheader=preheader,
            reply_to=reply_to,
            size_bytes=size_bytes,
            subject=subject,
            text=text,
            version=version,
            problem=problem,
        )

        return transactional_preview

