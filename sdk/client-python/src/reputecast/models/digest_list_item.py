from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_list_item_mode import DigestListItemMode
from ..models.digest_list_item_state import DigestListItemState
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.digest_issue_ref import DigestIssueRef
  from ..models.digest_person import DigestPerson
  from ..models.digest_schedule import DigestSchedule
  from ..models.digest_source_ref import DigestSourceRef





T = TypeVar("T", bound="DigestListItem")



@_attrs_define
class DigestListItem:
    """ 
        Attributes:
            id (str):
            mode (DigestListItemMode):
            name (str):
            schedule (DigestSchedule):
            source (DigestSourceRef):
            state (DigestListItemState):
            last (DigestIssueRef | Unset):
            next_ (DigestIssueRef | Unset):
            owner (DigestPerson | Unset):
     """

    id: str
    mode: DigestListItemMode
    name: str
    schedule: DigestSchedule
    source: DigestSourceRef
    state: DigestListItemState
    last: DigestIssueRef | Unset = UNSET
    next_: DigestIssueRef | Unset = UNSET
    owner: DigestPerson | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_issue_ref import DigestIssueRef # noqa: PLC0415
        from ..models.digest_person import DigestPerson # noqa: PLC0415
        from ..models.digest_schedule import DigestSchedule # noqa: PLC0415
        from ..models.digest_source_ref import DigestSourceRef # noqa: PLC0415
        id = self.id

        mode = self.mode.value

        name = self.name

        schedule = self.schedule.to_dict()

        source = self.source.to_dict()

        state = self.state.value

        last: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last, Unset):
            last = self.last.to_dict()

        next_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.next_, Unset):
            next_ = self.next_.to_dict()

        owner: dict[str, Any] | Unset = UNSET
        if not isinstance(self.owner, Unset):
            owner = self.owner.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "mode": mode,
            "name": name,
            "schedule": schedule,
            "source": source,
            "state": state,
        })
        if last is not UNSET:
            field_dict["last"] = last
        if next_ is not UNSET:
            field_dict["next"] = next_
        if owner is not UNSET:
            field_dict["owner"] = owner

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_issue_ref import DigestIssueRef # noqa: PLC0415
        from ..models.digest_person import DigestPerson # noqa: PLC0415
        from ..models.digest_schedule import DigestSchedule # noqa: PLC0415
        from ..models.digest_source_ref import DigestSourceRef # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        mode = DigestListItemMode(d.pop("mode"))




        name = d.pop("name")

        schedule = DigestSchedule.from_dict(d.pop("schedule"))




        source = DigestSourceRef.from_dict(d.pop("source"))




        state = DigestListItemState(d.pop("state"))




        _last = d.pop("last", UNSET)
        last: DigestIssueRef | Unset
        if isinstance(_last,  Unset):
            last = UNSET
        else:
            last = DigestIssueRef.from_dict(_last)




        _next_ = d.pop("next", UNSET)
        next_: DigestIssueRef | Unset
        if isinstance(_next_,  Unset):
            next_ = UNSET
        else:
            next_ = DigestIssueRef.from_dict(_next_)




        _owner = d.pop("owner", UNSET)
        owner: DigestPerson | Unset
        if isinstance(_owner,  Unset):
            owner = UNSET
        else:
            owner = DigestPerson.from_dict(_owner)




        digest_list_item = cls(
            id=id,
            mode=mode,
            name=name,
            schedule=schedule,
            source=source,
            state=state,
            last=last,
            next_=next_,
            owner=owner,
        )

        return digest_list_item

