from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.scoring_event_option import ScoringEventOption
  from ..models.scoring_field_option import ScoringFieldOption
  from ..models.scoring_form_option import ScoringFormOption
  from ..models.scoring_list_option import ScoringListOption





T = TypeVar("T", bound="ScoringOptions")



@_attrs_define
class ScoringOptions:
    """ 
        Attributes:
            api_events (list[ScoringEventOption]):
            fields (list[ScoringFieldOption]):
            forms (list[ScoringFormOption]):
            lists (list[ScoringListOption]):
     """

    api_events: list[ScoringEventOption]
    fields: list[ScoringFieldOption]
    forms: list[ScoringFormOption]
    lists: list[ScoringListOption]





    def to_dict(self) -> dict[str, Any]:
        from ..models.scoring_event_option import ScoringEventOption # noqa: PLC0415
        from ..models.scoring_field_option import ScoringFieldOption # noqa: PLC0415
        from ..models.scoring_form_option import ScoringFormOption # noqa: PLC0415
        from ..models.scoring_list_option import ScoringListOption # noqa: PLC0415
        api_events = []
        for api_events_item_data in self.api_events:
            api_events_item = api_events_item_data.to_dict()
            api_events.append(api_events_item)



        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "apiEvents": api_events,
            "fields": fields,
            "forms": forms,
            "lists": lists,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scoring_event_option import ScoringEventOption # noqa: PLC0415
        from ..models.scoring_field_option import ScoringFieldOption # noqa: PLC0415
        from ..models.scoring_form_option import ScoringFormOption # noqa: PLC0415
        from ..models.scoring_list_option import ScoringListOption # noqa: PLC0415
        d = dict(src_dict)
        api_events = []
        _api_events = d.pop("apiEvents")
        for api_events_item_data in (_api_events):
            api_events_item = ScoringEventOption.from_dict(api_events_item_data)



            api_events.append(api_events_item)


        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = ScoringFieldOption.from_dict(fields_item_data)



            fields.append(fields_item)


        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = ScoringFormOption.from_dict(forms_item_data)



            forms.append(forms_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ScoringListOption.from_dict(lists_item_data)



            lists.append(lists_item)


        scoring_options = cls(
            api_events=api_events,
            fields=fields,
            forms=forms,
            lists=lists,
        )

        return scoring_options

