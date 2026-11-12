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
  from ..models.page_person import PagePerson





T = TypeVar("T", bound="PageDraft")



@_attrs_define
class PageDraft:
    """ 
        Attributes:
            autosaved (bool):
            html (str):
            project (Any):
            saved_at (datetime.datetime):
            version (int):
            saved_by (PagePerson | Unset):
     """

    autosaved: bool
    html: str
    project: Any
    saved_at: datetime.datetime
    version: int
    saved_by: PagePerson | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_person import PagePerson # noqa: PLC0415
        autosaved = self.autosaved

        html = self.html

        project = self.project

        saved_at = self.saved_at.isoformat()

        version = self.version

        saved_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.saved_by, Unset):
            saved_by = self.saved_by.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "autosaved": autosaved,
            "html": html,
            "project": project,
            "savedAt": saved_at,
            "version": version,
        })
        if saved_by is not UNSET:
            field_dict["savedBy"] = saved_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_person import PagePerson # noqa: PLC0415
        d = dict(src_dict)
        autosaved = d.pop("autosaved")

        html = d.pop("html")

        project = d.pop("project")

        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        version = d.pop("version")

        _saved_by = d.pop("savedBy", UNSET)
        saved_by: PagePerson | Unset
        if isinstance(_saved_by,  Unset):
            saved_by = UNSET
        else:
            saved_by = PagePerson.from_dict(_saved_by)




        page_draft = cls(
            autosaved=autosaved,
            html=html,
            project=project,
            saved_at=saved_at,
            version=version,
            saved_by=saved_by,
        )

        return page_draft

