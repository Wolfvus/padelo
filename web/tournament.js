const sampleNames = ['Alex', 'Dani', 'Luz', 'Nico', 'Paz', 'Sol'];
// Each row holds the two doubles teams on court for one round.
const teams = [
  [[2, 3], [4, 5]],
  [[0, 4], [1, 5]],
  [[1, 2], [3, 5]],
  [[0, 2], [3, 4]],
  [[0, 5], [1, 3]],
  [[0, 1], [2, 4]],
];
const rounds = teams.map(([left, right]) => ({
  left, right,
  rest: sampleNames.map((_, index) => index).filter(index => !left.includes(index) && !right.includes(index)),
}));
const labels = {
  en: {
    skip: 'Skip to draw', language: 'Language', format: 'Proposed tournament format', title: '6 Loco · Six-player doubles',
    description: 'Six rounds on one court. Each player plays four matches with four different partners and rests twice.',
    edit: 'Edit player names', apply: 'Update draw', sample: 'Use sample names', draw: 'Round schedule',
    copy: 'Copy draw', manual: 'Select and copy the draw below.', copied: 'Draw copied.', updated: 'Draw updated.',
    player: 'Player', round: 'Round', vs: 'vs.', rest: 'Resting', empty: 'Enter a name for each player.',
    duplicate: 'Give each player a distinct name or nickname.',
  },
  es: {
    skip: 'Saltar a las rondas', language: 'Idioma', format: 'Propuesta de formato', title: '6 Loco · Dobles para seis',
    description: 'Seis rondas en una cancha. Cada persona juega cuatro partidos con cuatro compañeros distintos y descansa dos veces.',
    edit: 'Editar nombres', apply: 'Actualizar rondas', sample: 'Usar nombres de ejemplo', draw: 'Calendario de rondas',
    copy: 'Copiar rondas', manual: 'Selecciona y copia las rondas de abajo.', copied: 'Rondas copiadas.', updated: 'Rondas actualizadas.',
    player: 'Jugador', round: 'Ronda', vs: 'vs.', rest: 'Descansan', empty: 'Escribe un nombre para cada jugador.',
    duplicate: 'Usa un nombre o apodo distinto para cada jugador.',
  },
};
if (location.protocol === 'file:') document.querySelector('.back').href = 'https://github.com/Wolfvus/padelo';

const get = id => document.getElementById(id);
let language = navigator.language.toLowerCase().startsWith('es') ? 'es' : 'en';
let names = [...sampleNames];
let error = '';
let message = '';

function element(tag, className, text) {
  const node = document.createElement(tag);
  node.className = className;
  node.textContent = text;
  return node;
}

const inputs = names.map(name => {
  const label = element('label', '', '');
  const input = document.createElement('input');
  input.maxLength = 40;
  input.required = true;
  input.value = name;
  input.setAttribute('aria-describedby', 'error');
  label.append(element('span', 'player-label', ''), input);
  get('fields').append(label);
  return input;
});

function showText() {
  const t = labels[language];
  document.documentElement.lang = language;
  document.title = t.title + ' · Padelo';
  get('language').value = language;
  document.querySelectorAll('[data-text]').forEach(node => { node.textContent = t[node.dataset.text]; });
  document.querySelectorAll('.player-label').forEach((node, i) => { node.textContent = t.player + ' ' + (i + 1); });
  get('error').textContent = t[error] || '';
  get('status').textContent = t[message] || '';
}

function render() {
  showText();
  const t = labels[language];
  const cards = rounds.map(({left, right, rest}, i) => {
    const card = element('li', 'round', '');
    const match = element('div', 'match', '');
    for (const pair of [left, right]) {
      const team = element('div', 'team', '');
      for (const player of pair) team.append(element('span', '', names[player]));
      if (pair === right) match.append(element('span', 'versus', t.vs));
      match.append(team);
    }
    card.append(element('h3', '', t.round + ' ' + (i + 1)), match,
      element('p', 'rest', t.rest + ' · ' + rest.map(index => names[index]).join(' / ')));
    return card;
  });
  get('rounds').replaceChildren(...cards);
  get('manual').hidden = true;
  get('copy-text').value = '';
}

function applyNames(next) {
  names = next;
  error = '';
  message = 'updated';
  for (const input of inputs) input.removeAttribute('aria-invalid');
  get('roster').open = false;
  render();
  get('draw').focus();
}

get('names').addEventListener('submit', event => {
  event.preventDefault();
  const next = inputs.map(input => input.value.trim());
  const keys = next.map(name => name.normalize('NFC').toLocaleLowerCase(language));
  let bad = keys.findIndex(key => !key);
  error = 'empty';
  if (bad === -1) {
    bad = keys.findIndex((key, index) => keys.indexOf(key) !== index);
    error = 'duplicate';
  }
  if (bad === -1) return applyNames(next);
  inputs.forEach((input, index) => input.setAttribute('aria-invalid', String(index === bad)));
  showText();
  inputs[bad].focus();
});
get('sample').addEventListener('click', () => {
  inputs.forEach((input, index) => { input.value = sampleNames[index]; });
  applyNames([...sampleNames]);
});
get('language').addEventListener('change', event => {
  language = event.target.value;
  message = '';
  render();
});
get('copy').addEventListener('click', async () => {
  const t = labels[language];
  const pair = indices => indices.map(index => names[index]).join(' / ');
  const text = [t.title, ...rounds.map(({left, right, rest}, i) =>
    t.round + ' ' + (i + 1) + ' · ' + pair(left) + ' ' + t.vs + ' ' + pair(right) + '\n' + t.rest + ' · ' + pair(rest)
  )].join('\n\n');
  try {
    await navigator.clipboard.writeText(text);
    get('manual').hidden = true;
    message = 'copied';
  } catch {
    get('manual').hidden = false;
    get('copy-text').value = text;
    get('copy-text').focus();
    get('copy-text').select();
    message = '';
  }
  showText();
});
render();
