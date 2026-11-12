from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_check_group import ReadinessCheckGroup
from ..models.readiness_check_status import ReadinessCheckStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.readiness_check_row import ReadinessCheckRow
  from ..models.readiness_dismissal import ReadinessDismissal
  from ..models.readiness_fix import ReadinessFix
  from ..models.readiness_tech_row import ReadinessTechRow





T = TypeVar("T", bound="ReadinessCheck")



@_attrs_define
class ReadinessCheck:
    """ 
        Attributes:
            fixes (list[ReadinessFix]):
            flat_title (str):
            group (ReadinessCheckGroup):
            key (str):
            status (ReadinessCheckStatus):
            title (str):
            ack (str | Unset):
            ack_key (str | Unset):
            detail (str | Unset):
            dismissal (ReadinessDismissal | Unset):
            evidence (str | Unset):
            issue_keys (list[str] | Unset):
            label (str | Unset):
            mono (bool | Unset):
            note (str | Unset):
            raw (list[str] | Unset):
            rows (list[ReadinessCheckRow] | Unset):
            short (str | Unset):
            technical (list[ReadinessTechRow] | Unset):
     """

    fixes: list[ReadinessFix]
    flat_title: str
    group: ReadinessCheckGroup
    key: str
    status: ReadinessCheckStatus
    title: str
    ack: str | Unset = UNSET
    ack_key: str | Unset = UNSET
    detail: str | Unset = UNSET
    dismissal: ReadinessDismissal | Unset = UNSET
    evidence: str | Unset = UNSET
    issue_keys: list[str] | Unset = UNSET
    label: str | Unset = UNSET
    mono: bool | Unset = UNSET
    note: str | Unset = UNSET
    raw: list[str] | Unset = UNSET
    rows: list[ReadinessCheckRow] | Unset = UNSET
    short: str | Unset = UNSET
    technical: list[ReadinessTechRow] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_check_row import ReadinessCheckRow # noqa: PLC0415
        from ..models.readiness_dismissal import ReadinessDismissal # noqa: PLC0415
        from ..models.readiness_fix import ReadinessFix # noqa: PLC0415
        from ..models.readiness_tech_row import ReadinessTechRow # noqa: PLC0415
        fixes = []
        for fixes_item_data in self.fixes:
            fixes_item = fixes_item_data.to_dict()
            fixes.append(fixes_item)



        flat_title = self.flat_title

        group = self.group.value

        key = self.key

        status = self.status.value

        title = self.title

        ack = self.ack

        ack_key = self.ack_key

        detail = self.detail

        dismissal: dict[str, Any] | Unset = UNSET
        if not isinstance(self.dismissal, Unset):
            dismissal = self.dismissal.to_dict()

        evidence = self.evidence

        issue_keys: list[str] | Unset = UNSET
        if not isinstance(self.issue_keys, Unset):
            issue_keys = self.issue_keys



        label = self.label

        mono = self.mono

        note = self.note

        raw: list[str] | Unset = UNSET
        if not isinstance(self.raw, Unset):
            raw = self.raw



        rows: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.rows, Unset):
            rows = []
            for rows_item_data in self.rows:
                rows_item = rows_item_data.to_dict()
                rows.append(rows_item)



        short = self.short

        technical: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.technical, Unset):
            technical = []
            for technical_item_data in self.technical:
                technical_item = technical_item_data.to_dict()
                technical.append(technical_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fixes": fixes,
            "flatTitle": flat_title,
            "group": group,
            "key": key,
            "status": status,
            "title": title,
        })
        if ack is not UNSET:
            field_dict["ack"] = ack
        if ack_key is not UNSET:
            field_dict["ackKey"] = ack_key
        if detail is not UNSET:
            field_dict["detail"] = detail
        if dismissal is not UNSET:
            field_dict["dismissal"] = dismissal
        if evidence is not UNSET:
            field_dict["evidence"] = evidence
        if issue_keys is not UNSET:
            field_dict["issueKeys"] = issue_keys
        if label is not UNSET:
            field_dict["label"] = label
        if mono is not UNSET:
            field_dict["mono"] = mono
        if note is not UNSET:
            field_dict["note"] = note
        if raw is not UNSET:
            field_dict["raw"] = raw
        if rows is not UNSET:
            field_dict["rows"] = rows
        if short is not UNSET:
            field_dict["short"] = short
        if technical is not UNSET:
            field_dict["technical"] = technical

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_check_row import ReadinessCheckRow # noqa: PLC0415
        from ..models.readiness_dismissal import ReadinessDismissal # noqa: PLC0415
        from ..models.readiness_fix import ReadinessFix # noqa: PLC0415
        from ..models.readiness_tech_row import ReadinessTechRow # noqa: PLC0415
        d = dict(src_dict)
        fixes = []
        _fixes = d.pop("fixes")
        for fixes_item_data in (_fixes):
            fixes_item = ReadinessFix.from_dict(fixes_item_data)



            fixes.append(fixes_item)


        flat_title = d.pop("flatTitle")

        group = ReadinessCheckGroup(d.pop("group"))




        key = d.pop("key")

        status = ReadinessCheckStatus(d.pop("status"))




        title = d.pop("title")

        ack = d.pop("ack", UNSET)

        ack_key = d.pop("ackKey", UNSET)

        detail = d.pop("detail", UNSET)

        _dismissal = d.pop("dismissal", UNSET)
        dismissal: ReadinessDismissal | Unset
        if isinstance(_dismissal,  Unset):
            dismissal = UNSET
        else:
            dismissal = ReadinessDismissal.from_dict(_dismissal)




        evidence = d.pop("evidence", UNSET)

        issue_keys = cast(list[str], d.pop("issueKeys", UNSET))


        label = d.pop("label", UNSET)

        mono = d.pop("mono", UNSET)

        note = d.pop("note", UNSET)

        raw = cast(list[str], d.pop("raw", UNSET))


        _rows = d.pop("rows", UNSET)
        rows: list[ReadinessCheckRow] | Unset = UNSET
        if _rows is not UNSET:
            rows = []
            for rows_item_data in _rows:
                rows_item = ReadinessCheckRow.from_dict(rows_item_data)



                rows.append(rows_item)


        short = d.pop("short", UNSET)

        _technical = d.pop("technical", UNSET)
        technical: list[ReadinessTechRow] | Unset = UNSET
        if _technical is not UNSET:
            technical = []
            for technical_item_data in _technical:
                technical_item = ReadinessTechRow.from_dict(technical_item_data)



                technical.append(technical_item)


        readiness_check = cls(
            fixes=fixes,
            flat_title=flat_title,
            group=group,
            key=key,
            status=status,
            title=title,
            ack=ack,
            ack_key=ack_key,
            detail=detail,
            dismissal=dismissal,
            evidence=evidence,
            issue_keys=issue_keys,
            label=label,
            mono=mono,
            note=note,
            raw=raw,
            rows=rows,
            short=short,
            technical=technical,
        )

        return readiness_check

